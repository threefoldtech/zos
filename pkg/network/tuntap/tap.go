package tuntap

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

const disableIPv6Template = "net.ipv6.conf.%s.disable_ipv6"

// CreateTap creates a new tap device with the given name, sets the master interface and moves it
// into the given network namespace.
func CreateTap(name string, master string, netns ns.NetNS) (*netlink.Tuntap, error) {
	masterIface, err := netlink.LinkByName(master)
	if err != nil {
		return nil, errors.Wrap(err, "failed to look up tap master")
	}

	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			MTU:         1500,
			Name:        name,
			ParentIndex: masterIface.Attrs().Index,
			Namespace:   netlink.NsFd(int(netns.Fd())),
		},
		Mode: netlink.TUNTAP_MODE_TAP,
	}

	if err := netlink.LinkAdd(tap); err != nil {
		return nil, errors.Wrap(err, "could not add tap device")
	}

	err = netns.Do(func(_ ns.NetNS) error {
		disableIPv6Cmd := fmt.Sprintf(disableIPv6Template, name)
		if _, err := sysctl.Sysctl(disableIPv6Cmd, "1"); err != nil {
			_ = netlink.LinkDel(tap)
			return errors.Wrap(err, "failed to disable ipv6 on interface host side")
		}

		// Re-fetch tap to get all properties/attributes
		link, err := netlink.LinkByName(name)
		if err != nil {
			return errors.Wrapf(err, "failed to refetch tap interface %q", name)
		}

		var ok bool
		tap, ok = link.(*netlink.Tuntap)
		if !ok {
			return fmt.Errorf("link %s should be of type tuntap", name)
		}

		if tap.Mode != netlink.TUNTAP_MODE_TAP {
			return errors.New("tuntap iface does not have the expected 'tap' mode")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return tap, nil
}
