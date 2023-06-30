package networkd

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"net"
	"os/exec"
	"text/template"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

//go:embed nft/script.sh
var script embed.FS

// getHomeNetwork is a helper function that returns the physical interface
// used by zos bridge and the mac address of the gw. this is needed to
func getHomeNetwork() (link netlink.Link, gw *net.HardwareAddr, err error) {
	master, err := netlink.LinkByName(types.DefaultBridge)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get default bridge")
	}

	routes, err := netlink.RouteList(master, netlink.FAMILY_V4)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get current routing table")
	}

	for _, route := range routes {
		if route.Dst != nil {
			continue
		}
		neighbors, err := netlink.NeighList(master.Attrs().Index, netlink.FAMILY_V4)

		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to list neighbors")
		}

		for _, neigh := range neighbors {
			if neigh.IP.Equal(route.Gw) {
				gw = &neigh.HardwareAddr
				break
			}

		}

		break
	}

	if gw == nil {
		// no default route !
		return nil, nil, fmt.Errorf("failed to get mac of default gw")
	}

	all, err := netlink.LinkList()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to list links")
	}

	for _, dev := range all {
		if dev.Type() == "device" && dev.Attrs().MasterIndex == master.Attrs().Index {
			link = dev
			break
		}
	}

	if link == nil {
		return nil, nil, fmt.Errorf("failed to find exit device for home network")
	}

	return link, gw, nil
}

func ensureHostFw(ctx context.Context) error {

	exit, gw, err := getHomeNetwork()
	if err != nil {
		return err
	}

	log.Info().Str("device", exit.Attrs().Name).Str("gw", gw.String()).Msg("home network")
	log.Info().Msg("ensuring existing host nft rules")

	tmp, err := template.ParseFS(script, "nft/script.sh")
	if err != nil {
		return err
	}

	input := struct {
		Inf     string
		Gateway string
	}{
		Inf:     exit.Attrs().Name,
		Gateway: gw.String(),
	}

	var buf bytes.Buffer
	if err := tmp.Execute(&buf, input); err != nil {
		return errors.Wrap(err, "failed to render nft script template")
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", buf.String())

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "could not set up host nft rules")
	}

	return nil
}
