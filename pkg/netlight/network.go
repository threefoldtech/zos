package netlight

import (
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/netlight/bridge"
	"github.com/threefoldtech/zos/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zos/pkg/netlight/macvlan"
	"github.com/threefoldtech/zos/pkg/netlight/options"
	"github.com/vishvananda/netlink"
)

const (
	NDMZBridge = "br-ndmz"
	NDMZGw     = "gw"
)

var (
	NDMZGwIP = &net.IPNet{
		IP:   net.ParseIP("100.127.0.1"),
		Mask: net.CIDRMask(16, 32),
	}
)

func CreateNDMZBridge() (*netlink.Bridge, error) {
	return createNDMZBridge(NDMZBridge, NDMZGw)
}

func createNDMZBridge(name string, gw string) (*netlink.Bridge, error) {
	if !bridge.Exists(name) {
		if _, err := bridge.New(name); err != nil {
			return nil, errors.Wrapf(err, "couldn't create bridge %s", name)
		}
	}

	if err := options.Set(name, options.IPv6Disable(true)); err != nil {
		return nil, errors.Wrapf(err, "failed to disable ip6 on bridge %s", name)
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get ndmz bridge: %w", err)
	}

	if link.Type() != "bridge" {
		return nil, fmt.Errorf("ndmz is not a bridge")
	}

	if !ifaceutil.Exists(gw, nil) {
		gwLink, err := macvlan.Create(gw, name, nil)
		if err != nil {
			return nil, err
		}

		addrs := []*netlink.Addr{
			{
				IPNet: NDMZGwIP,
			},
		}

		for _, addr := range addrs {
			err = netlink.AddrAdd(gwLink, addr)
			if err != nil && !os.IsExist(err) {
				return nil, err
			}
		}

		if err := netlink.LinkSetUp(gwLink); err != nil {
			return nil, err
		}

	}

	if err := netlink.LinkSetUp(link); err != nil {
		return nil, err
	}

	return link.(*netlink.Bridge), nil
}
