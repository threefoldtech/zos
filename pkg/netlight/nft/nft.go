package nft

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"github.com/pkg/errors"
)

// Apply applies the ntf configuration contained in the reader r
// if ns is specified, the nft command is execute in the network namespace names ns
func Apply(r io.Reader, ns string) error {
	var cmd *exec.Cmd

	if ns != "" {
		cmd = exec.Command("ip", "netns", "exec", ns, "nft", "-f", "-")
	} else {
		cmd = exec.Command("nft", "-f", "-")
	}

	cmd.Stdin = r

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(out)).Msg("error during nft")
		if eerr, ok := err.(*exec.ExitError); ok {
			return errors.Wrapf(err, "failed to execute nft: %v", string(eerr.Stderr))
		}
		return errors.Wrap(err, "failed to execute nft")
	}
	return nil
}

func flushChain(ns, table, chain string) error {
	args := []string{"flush", "chain", table, chain}
	var cmd *exec.Cmd

	if ns != "" {
		cmd = exec.Command("ip", "netns", "exec", ns, "nft")
		cmd.Args = append(cmd.Args, args...)
	} else {
		cmd = exec.Command("nft", args...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to flush nft chain: %v", string(out))
	}

	return nil
}

// DropTrafficToLAN drops all the outgoing traffic to any peers on
// the same lan network, but allow dicovery port for ygg/myc by accepting
// traffic to/from dest/src ports.
// @th,0,16 and @th,16,16 is raw expression for sport/dport in transport header
// used due to limitation on the installed nft v0.9.1
func DropTrafficToLAN() error {
	dgw, err := getDefaultGW()
	if err != nil {
		return fmt.Errorf("failed to find default gateway: %w", err)
	}

	// apply anyway even the if the gateway is private; to clean the previous rules
	if err := flushChain("", "inet filter", "forward"); err != nil {
		return err
	}

	if !dgw.IP.IsPrivate() {
		log.Warn().Msg("skip LAN security. default gateway is public")
		return nil
	}

	ipAddr := dgw.IP.String()
	netAddr := getNetworkRange(dgw)
	macAddr := dgw.HardwareAddr.String()
	log.Debug().
		Str("ipAddr", ipAddr).
		Str("netAddr", netAddr).
		Str("macAddr", macAddr).
		Msg("drop traffic to lan with the default gateway")

	var buf bytes.Buffer
	buf.WriteString("table inet filter {\n")
	buf.WriteString("  chain forward {\n")
	// allow traffic on sport ygg/myc discovery ports
	buf.WriteString("    meta l4proto { tcp, udp } @th,0,16 { 9651, 9650 } accept;\n")
	// allow traffic on dport ygg/myc discovery ports
	buf.WriteString("    meta l4proto { tcp, udp } @th,16,16 { 9651, 9650 } accept;\n")
	// allow traffic to the default gateway
	buf.WriteString(fmt.Sprintf("    ip daddr %s accept;\n", ipAddr))
	// allow traffic to any ip not in the network range
	buf.WriteString(fmt.Sprintf("    ip daddr != %s accept;\n", netAddr))
	// only drop traffic if it destined to mac addr other than the default gateway
	buf.WriteString(fmt.Sprintf("    ether daddr != %s drop;\n", macAddr))
	buf.WriteString("  }\n")
	buf.WriteString("}\n")

	// applied on host
	return Apply(&buf, "")
}

func getDefaultGW() (netlink.Neigh, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return netlink.Neigh{}, fmt.Errorf("failed to list routes: %v", err)
	}

	var defaultRoute *netlink.Route
	for _, route := range routes {
		if route.Dst == nil {
			defaultRoute = &route
			break
		}
	}

	if defaultRoute == nil {
		return netlink.Neigh{}, fmt.Errorf("default route not found")
	}

	if defaultRoute.Gw == nil {
		return netlink.Neigh{}, fmt.Errorf("default route has no gateway")
	}

	neighs, err := netlink.NeighList(0, netlink.FAMILY_V4)
	if err != nil {
		return netlink.Neigh{}, fmt.Errorf("failed to list neighbors: %v", err)
	}

	for _, neigh := range neighs {
		if neigh.IP.Equal(defaultRoute.Gw) {
			return neigh, nil
		}
	}

	return netlink.Neigh{}, errors.New("failed to get default gw")
}

func getNetworkRange(ip netlink.Neigh) string {
	mask := ip.IP.DefaultMask()
	network := ip.IP.Mask(mask)
	ones, _ := mask.Size()
	networkRange := fmt.Sprintf("%s/%d", network.String(), ones)

	return networkRange
}
