package nft

import (
	"fmt"
	"io"
	"net"
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

// DropTrafficToLAN drops all the outgoing traffic to any peers on
// the same lan network
func DropTrafficToLAN() error {
	mac, err := getDefaultGwMac()
	log.Debug().Str("mac", mac.String()).Err(err).Msg("default gw return")

	cmd := exec.Command("nft", "add", "rule", "inet", "filter", "forward",
		"ether", "daddr", "!=", mac.String(), "drop")

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

func getDefaultGwMac() (net.HardwareAddr, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %v", err)
	}

	var defaultRoute *netlink.Route
	for _, route := range routes {
		if route.Dst == nil {
			defaultRoute = &route
			break
		}
	}

	if defaultRoute == nil {
		return nil, fmt.Errorf("default route not found")
	}

	if defaultRoute.Gw == nil {
		return nil, fmt.Errorf("default route has no gateway")
	}

	neighs, err := netlink.NeighList(0, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to list neighbors: %v", err)
	}

	for _, neigh := range neighs {
		if neigh.IP.Equal(defaultRoute.Gw) {
			return neigh.HardwareAddr, nil
		}
	}

	return nil, errors.New("failed to get default gw")
}
