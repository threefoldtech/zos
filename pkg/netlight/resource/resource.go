package resource

import (
	"embed"
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/netlight/bridge"
	"github.com/threefoldtech/zos/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zos/pkg/netlight/macvlan"
	"github.com/threefoldtech/zos/pkg/netlight/namespace"
	"github.com/threefoldtech/zos/pkg/network/nft"
	"github.com/vishvananda/netlink"
)

const (
	INF_PUBLIC   = "public"
	INF_PRIVATE  = "private"
	INF_MYCELIUM = "mycelium"
)

//go:embed nft/rules.nft
var nftRules embed.FS

// Create creates a network resource (please check docs)
// name: is the name of the network resource. The Create function is idempotent which means if the same name is used the function
// will not recreate the resource.
// master: Normally the br-ndmz bridge, this is the resource "way out" to the public internet. A `public` interface is created and wired
// to the master bridge
// ndmzIP: the IP assigned to the `public` interface.
// ndmzGwIP: the gw Ip for the resource. Normally this is the ip assigned to the master bridge.
// privateNet: optional private network range
// seed: mycelium seed
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
		if privateNet.IP.To4() == nil {
			return fmt.Errorf("private ip range must be IPv4 address")
		}

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
		if err := ifaceutil.SetLoUp(); err != nil {
			return fmt.Errorf("failed to set lo up for namespace '%s': %w", nsName, err)
		}

		if err := setLinkAddr(INF_PUBLIC, ndmzIP); err != nil {
			return fmt.Errorf("couldn't set link addr for public interface in namespace %s: %w", nsName, err)
		}

		if privateNet != nil {
			privGwIp := privateNet.IP.To4()
			// this is to take the first IP of the private range
			// and use it as the gw address for the entire private range
			privGwIp[net.IPv4len-1] = 1
			// this IP is then set on the private interface
			privateNet.IP = privGwIp
			if err := setLinkAddr(INF_PRIVATE, privateNet); err != nil {
				return fmt.Errorf("couldn't set link addr for private interface in namespace %s: %w", nsName, err)
			}
		}

		// if err := setLinkAddr(INF_PRIVATE, )
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

	rules, err := nftRules.Open("nft/rules.nft")
	if err != nil {
		return fmt.Errorf("failed to load nft.rules file")
	}

	if err := nft.Apply(rules, nsName); err != nil {
		return fmt.Errorf("failed to apply nft rules for namespace '%s': %w", name, err)
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
