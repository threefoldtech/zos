package macvlan

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/options"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
)

// Create creates a new macvlan interface in the network namespace
// name is the name of the macvlan interface
// master is the name of the device used as master for the macvlan interface
// netns is network namespace where to create the macvlan
func Create(name string, master string, netns ns.NetNS) (*netlink.Macvlan, error) {

	m, err := netlink.LinkByName(master)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup master %q: %v", master, err)
	}

	// due to kernel bug we have to create with tmpName or it might
	// collide with the name on the host and error out
	tmpName, err := ip.RandomVethName()
	if err != nil {
		return nil, err
	}

	mv := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			MTU:         1500,
			Name:        tmpName,
			ParentIndex: m.Attrs().Index,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}

	if netns != nil {
		mv.Namespace = netlink.NsFd(int(netns.Fd()))
	}

	if err := netlink.LinkAdd(mv); err != nil {
		return nil, fmt.Errorf("failed to create macvlan: %v", err)
	}

	f := func(_ ns.NetNS) error {
		// TODO: duplicate following lines for ipv6 support, when it will be added in other places

		// containernetworking sets it up in some ref code somewhere, we copied it. we were stoopit
		// disable proxy_arp, as it is ony useful in some very distinct cases, and otherwise can wreak
		// havoc in networks -> 0!
		if err := options.Set(tmpName, options.ProxyArp(false)); err != nil {
			// remove the newly added link and ignore errors, because we already are in a failed state
			_ = netlink.LinkDel(mv)
			return fmt.Errorf("failed to set proxy_arp on newly added interface %q: %v", tmpName, err)
		}

		err := ip.RenameLink(tmpName, name)
		if err != nil {
			_ = netlink.LinkDel(mv)
			return fmt.Errorf("failed to rename macvlan to %q: %v", name, err)
		}

		// Re-fetch macvlan to get all properties/attributes
		link, err := netlink.LinkByName(name)
		if err != nil {
			return fmt.Errorf("failed to refetch macvlan %q: %v", name, err)
		}
		var ok bool
		mv, ok = link.(*netlink.Macvlan)
		if !ok {
			return fmt.Errorf("link %s should be of type macvlan", name)
		}

		return nil
	}
	if netns != nil {
		err = netns.Do(f)
	} else {
		err = f(nil)
	}

	return mv, err
}

// Install configures a macvlan interfaces created with Create method
func Install(link *netlink.Macvlan, hw net.HardwareAddr, ips []*net.IPNet, routes []*netlink.Route, netns ns.NetNS) error {
	f := func(_ ns.NetNS) error {
		if len(hw) != 0 {
			if !bytes.Equal(link.HardwareAddr, hw) {
				if err := netlink.LinkSetHardwareAddr(link, hw); err != nil {
					return fmt.Errorf("failed to set MAC address on interface %s: %w", link.Attrs().Name, err)
				}
			}
		}

		name := link.Attrs().Name
		ipsMap := make(map[string]struct{})
		routesMap := make(map[string]struct{})
		for _, ip := range ips {
			ipsMap[ip.String()] = struct{}{}
		}

		for _, route := range routes {
			routesMap[route.String()] = struct{}{}
		}
		if current, err := netlink.AddrList(link, netlink.FAMILY_ALL); err == nil {
			for _, addr := range current {
				if _, ok := ipsMap[addr.IPNet.String()]; ok {
					// only delete ips that are not to be installed
					continue
				}
				if err := netlink.AddrDel(link, &addr); err != nil {
					log.Error().Err(err).Str("ip", addr.IP.String()).Msg("failed to delete address")
				}
			}
		}

		// NOTE: this is dangerous because it also deletes default network routes. this then must
		// be rewritten.
		// if current, err := netlink.RouteList(link, netlink.FAMILY_ALL); err == nil {
		// 	for _, route := range current {
		// 		if _, ok := routesMap[route.String()]; ok {
		// 			// only delete routes that are not to be installed
		// 			continue
		// 		}
		// 		if err := netlink.RouteDel(&route); err != nil {
		// 			log.Error().Err(err).Str("route", route.String()).Msg("failed to delete route")
		// 		}
		// 	}
		// }

		for _, ip := range ips {
			if err := netlink.AddrAdd(link, &netlink.Addr{
				IPNet: ip,
			}); err != nil && !os.IsExist(err) {
				log.Error().
					Str("addr", ip.String()).
					Str("link", link.Attrs().Name).
					Err(err).Msg("failed to set address on link")
				return err
			}
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to set %q UP: %v", name, err)
		}

		for _, route := range routes {
			route.LinkIndex = link.Attrs().Index

			if err := netlink.RouteAdd(route); err != nil && !os.IsExist(err) {
				log.Error().
					Str("route", route.String()).
					Str("link", link.Attrs().Name).
					Err(err).Msg("failed to set route on link")
				return err
			}
		}

		return nil
	}
	if netns != nil {
		return netns.Do(f)
	}

	return f(nil)
}

// GetByName return a macvlan object by its name
func GetByName(name string) (*netlink.Macvlan, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	macvlan, ok := link.(*netlink.Macvlan)
	if !ok {
		return nil, fmt.Errorf("link %s is not a macvlan", name)
	}
	return macvlan, nil
}
