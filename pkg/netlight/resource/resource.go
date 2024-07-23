package resource

import (
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/netlight/bridge"
	"github.com/threefoldtech/zos/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zos/pkg/netlight/macvlan"
	"github.com/threefoldtech/zos/pkg/netlight/namespace"
	"github.com/vishvananda/netlink"
)

const (
	INF_PUBLIC   = "public"
	INF_PRIVATE  = "private"
	INF_MYCELIUM = "mycelium"
)

// type Resource struct {
// 	name string
// }

// Create creates a network name space and wire it to the master bridge
func Create(name string, master *netlink.Bridge, ndmzIP *net.IPNet, ndmzGwIP *net.IPNet, privateNet *net.IPNet, seed []byte) error {
	privateNetBr := fmt.Sprintf("r%s", name)
	myBr := fmt.Sprintf("m%s", name)
	nsName := fmt.Sprintf("n%s", name)

	bridges := []string{myBr}
	if privateNet != nil {
		bridges = append(bridges, privateNetBr)
	}

	for _, name := range bridges {
		if !bridge.Exists(name) {
			if _, err := bridge.New(name); err != nil {
				return fmt.Errorf("couldn't create bridge %s: %w", name, err)
			}
		}
	}

	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		netNS, err = namespace.Create(nsName)
		if err != nil {
			return fmt.Errorf("failed to create namespace %s", err)
		}
	}

	defer netNS.Close()

	if privateNet != nil {
		if !ifaceutil.Exists(INF_PRIVATE, netNS) {
			if _, err = macvlan.Create(INF_PRIVATE, privateNetBr, netNS); err != nil {
				return fmt.Errorf("failed to create private link: %w", err)
			}
		}
	}

	// create public interface and attach it to ndmz bridge
	if !ifaceutil.Exists(INF_PUBLIC, netNS) {
		if _, err = macvlan.Create(INF_PUBLIC, master.Name, netNS); err != nil {
			return fmt.Errorf("failed to create public link: %w", err)
		}
	}

	if !ifaceutil.Exists(INF_MYCELIUM, netNS) {
		if _, err = macvlan.Create(INF_MYCELIUM, myBr, netNS); err != nil {
			return err
		}
	}

	err = netNS.Do(func(_ ns.NetNS) error {
		if err := setLinkAddr(INF_PUBLIC, ndmzIP); err != nil {
			return fmt.Errorf("couldn't set link addr for public interface in namespace %s: %w", nsName, err)
		}

		if err := netlink.RouteAdd(&netlink.Route{
			Gw: ndmzGwIP.IP,
		}); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to add ndmz routing")
		}

		return nil
	})

	if err != nil {
		return err
	}

	return setupMycelium(netNS, INF_MYCELIUM, seed)
}

func setLinkAddr(name string, ip *net.IPNet) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to set link address: %w", err)
	}

	// if err := options.Set(name, options.IPv6Disable(false)); err != nil {
	// 	return fmt.Errorf("failed to enable ip6 on interface %s: %w", name, err)
	// }

	addr := netlink.Addr{
		IPNet: ip,
	}
	err = netlink.AddrAdd(link, &addr)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to add ip address to link: %w", err)
	}

	return netlink.LinkSetUp(link)
}
