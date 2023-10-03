package perf

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

const macvlanName = "pubtestmacvlan"
const namespaceName = "pubtestns"

type PubIPTask struct {
	taskID   string
	schedule string
}

var _ Task = (*PubIPTask)(nil)

func (p *PubIPTask) ID() string {
	return p.taskID
}

func (p *PubIPTask) Cron() string {
	return p.schedule
}

func (p *PubIPTask) Run(ctx context.Context) (interface{}, error) {
	netNS, err := namespace.GetByName(namespaceName)
	if err != nil {
		netNS, err = namespace.Create(namespaceName)
		if err != nil {
			return nil, fmt.Errorf("failed to create namespace %s: %w", namespaceName, err)
		}
	}
	mv, err := macvlan.GetByName(macvlanName)
	if err != nil {
		mv, err = macvlan.Create(macvlanName, types.PublicBridge, netNS)
		if err != nil {
			return nil, fmt.Errorf("failed to create macvlan %s: %w", namespaceName, err)
		}
	}
	manager, err := environment.GetSubstrate()
	if err != nil {
		return nil, fmt.Errorf("failed to get substrate client: %w", err)
	}
	sub, err := manager.Substrate()
	if err != nil {
		return nil, fmt.Errorf("failed to get substrate client: %w", err)
	}
	farmID := environment.MustGet().FarmID
	farm, err := sub.GetFarm(uint32(farmID))
	if farm != nil {
		return nil, fmt.Errorf("failed to get farm with id %d: %w", farmID, err)
	}
	unusedIPs := map[string]bool{}
	for _, publicIP := range farm.PublicIPs {
		if publicIP.ContractID == 0 {
			unusedIPs[publicIP.IP] = false
		}
	}
	for ip := range unusedIPs {
		_, ipNet, err := net.ParseCIDR(ip)
		if err != nil {
			continue
		}
		err = macvlan.Install(mv, nil, []*net.IPNet{ipNet}, nil, netNS)
		if err != nil {
			return nil, fmt.Errorf("failed to install macvlan %s with ip %s to namespace %s: %w", macvlanName, ipNet, namespaceName, err)
		}
		err = netNS.Do(func(_ ns.NetNS) error {
			req, err := http.Get("https://api.ipify.org/")
			if err != nil {
				return err
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				req.Body.Close()
				return err
			}
			req.Body.Close()

			if ip == strings.TrimSpace(string(body)) {
				unusedIPs[ip] = true
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check if ip is valid: %w", err)
		}

	}
	err = netlink.LinkSetDown(mv)
	if err != nil {
		return nil, fmt.Errorf("failed to set macvlan %s link down: %w", macvlanName, err)
	}

	return unusedIPs, nil
}
