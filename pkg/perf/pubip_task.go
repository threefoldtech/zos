package perf

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/vishvananda/netlink"
)

const testMacvlan = "pubtestmacvlan"
const testNamespace = "pubtestns"

type publicIPValidationTask struct {
	taskID    string
	schedule  string
	unusedIPs map[string]bool
	publicIPs []substrate.PublicIP
}

var _ Task = (*publicIPValidationTask)(nil)

func NewPublicIPValidationTask() Task {
	return &publicIPValidationTask{
		taskID:    "PublicIPValidation",
		schedule:  "0 0 */6 * * *",
		unusedIPs: make(map[string]bool),
		publicIPs: make([]substrate.PublicIP, 0),
	}
}

func (p *publicIPValidationTask) ID() string {
	return p.taskID
}

func (p *publicIPValidationTask) Cron() string {
	return p.schedule
}

func (p *publicIPValidationTask) Run(ctx context.Context) (interface{}, error) {

	netNS, err := namespace.GetByName(testNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", testNamespace, err)
	}
	manager, err := environment.GetSubstrate()
	if err != nil {
		return nil, fmt.Errorf("failed to get substrate client: %w", err)
	}
	sub, err := manager.Substrate()
	if err != nil {
		return nil, fmt.Errorf("failed to get substrate client: %w", err)
	}
	defer sub.Close()
	farmID := environment.MustGet().FarmID

	shouldRun, err := isLeastValidNode(ctx, uint32(farmID), sub)
	if err != nil {
		return nil, fmt.Errorf("failed to check if the node should run public IP verification: %w", err)
	}
	if !shouldRun {
		log.Info().Msg("skipping because there is a node with less ID available")
		return nil, nil
	}

	farm, err := sub.GetFarm(uint32(farmID))
	if err != nil {
		return nil, fmt.Errorf("failed to get farm with id %d: %w", farmID, err)
	}
	p.publicIPs = farm.PublicIPs
	err = netNS.Do(p.validateIPs)

	if err != nil {
		return nil, fmt.Errorf("failed to run public IP validation: %w", err)
	}
	return p.unusedIPs, nil
}

func (p *publicIPValidationTask) validateIPs(_ ns.NetNS) error {
	mv, err := macvlan.GetByName(testMacvlan)
	if err != nil {
		return fmt.Errorf("failed to get macvlan %s in namespace %s: %w", testMacvlan, testNamespace, err)
	}
	// to delete any leftover IPs or routes
	err = deleteAllIPsAndRoutes(mv)
	if err != nil {
		log.Err(err).Send()
	}
	for _, publicIP := range p.publicIPs {
		if publicIP.ContractID != 0 {
			continue
		}
		p.unusedIPs[publicIP.IP] = false

		ip, ipNet, routes, err := getIPWithRoute(publicIP)
		if err != nil {
			log.Err(err).Send()
			continue
		}
		err = macvlan.Install(mv, nil, ipNet, routes, nil)
		if err != nil {
			log.Err(err).Msgf("failed to install macvlan %s with ip %s to namespace %s", testMacvlan, ipNet, testNamespace)
			continue
		}

		realIP, err := getRealPublicIP()
		if err != nil {
			log.Err(err).Msg("failed to get node real IP")
		}

		if ip.String() == strings.TrimSpace(string(realIP)) {
			p.unusedIPs[publicIP.IP] = true
		}

		err = deleteAllIPsAndRoutes(mv)
		if err != nil {
			log.Err(err).Send()
		}
	}
	err = netlink.LinkSetDown(mv)
	if err != nil {
		return fmt.Errorf("failed to set link down: %w", err)
	}
	return nil
}

func isLeastValidNode(ctx context.Context, farmID uint32, sub *substrate.Substrate) (bool, error) {
	nodes, err := sub.GetNodes(uint32(farmID))
	if err != nil {
		return false, fmt.Errorf("failed to get farm %d nodes: %w", farmID, err)
	}
	cl := GetZbusClient(ctx)
	registrar := stubs.NewRegistrarStub(cl)
	var nodeID uint32
	err = backoff.Retry(func() error {
		nodeID, err = registrar.NodeID(ctx)
		if err != nil {
			log.Err(err).Msg("failed to get node id")
			return err
		}
		return nil
	}, backoff.NewConstantBackOff(10*time.Second))

	if err != nil {
		return false, fmt.Errorf("failed to get node id: %w", err)
	}

	for _, node := range nodes {
		if node >= uint32(nodeID) {
			continue
		}
		state, err := sub.GetPowerTarget(node)
		if err != nil {
			return false, fmt.Errorf("failed to get node %d power target: %w", node, err)
		}
		if state.Target.IsDown {
			continue
		}
		n, err := sub.GetNode(node)
		if err != nil {
			return false, fmt.Errorf("failed to get node %d: %w", node, err)
		}
		ip, err := getValidNodeIP(n)
		if err != nil {
			return false, err
		}
		// stop at three and quiet output
		err = exec.CommandContext(ctx, "ping", "-c", "3", "-q", ip).Run()
		if err != nil {
			log.Err(err).Msgf("failed to ping node %d", node)
			continue
		}
		return false, nil
	}
	return true, nil
}

func getValidNodeIP(node *substrate.Node) (string, error) {
	for _, inf := range node.Interfaces {
		if inf.Name != "zos" {
			continue
		}
		if len(inf.IPs) == 0 {
			return "", fmt.Errorf("no private IP available on node %d", node.ID)
		}
		return inf.IPs[0], nil
	}
	return "", fmt.Errorf("failed to get private IP for node %d", node.ID)
}

func getIPWithRoute(publicIP substrate.PublicIP) (net.IP, []*net.IPNet, []*netlink.Route, error) {
	ip, ipNet, err := net.ParseCIDR(publicIP.IP)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse IP %s: %w", publicIP.IP, err)
	}
	gateway := net.ParseIP(publicIP.Gateway)
	if gateway == nil {
		return nil, nil, nil, fmt.Errorf("failed to parse gateway %s: %w", publicIP.Gateway, err)
	}
	route, err := netlink.RouteGet(gateway)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get route to gateway %s", publicIP.Gateway)
	}
	routes := make([]*netlink.Route, 0)
	for _, r := range route {
		routes = append(routes, &r)
	}
	return ip, []*net.IPNet{ipNet}, routes, nil
}

func getRealPublicIP() (string, error) {
	// for testing now, should change to cloudflare
	req, err := http.Get("https://api.ipify.org/")
	if err != nil {
		return "", err
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func deleteAllIPsAndRoutes(macvlan netlink.Link) error {
	addresses, err := netlink.AddrList(macvlan, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to list addresses in macvlan %s: %w", testMacvlan, err)
	}
	for _, addr := range addresses {
		err = netlink.AddrDel(macvlan, &addr)
		if err != nil {
			log.Err(err).Msgf("failed to delete address %s", addr)
		}
	}
	routes, err := netlink.RouteList(macvlan, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to list routes in macvlan %s: %w", testMacvlan, err)
	}
	for _, route := range routes {
		err = netlink.RouteDel(&route)
		if err != nil {
			log.Err(err).Msgf("failed to delete route %s", route)
		}
	}
	return err
}
