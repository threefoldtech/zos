package resource

import (
	"crypto/rand"
	"embed"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/netlight/bridge"
	"github.com/threefoldtech/zos/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zos/pkg/netlight/namespace"
	"github.com/threefoldtech/zos/pkg/netlight/nft"
	"github.com/threefoldtech/zos/pkg/netlight/options"
	"github.com/threefoldtech/zos/pkg/netlight/tuntap"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/vishvananda/netlink"
)

const (
	infPublic   = "public"
	infPrivate  = "private"
	infMycelium = "mycelium"
)

//go:embed nft/rules.nft
var nftRules embed.FS

type Resource struct {
	name string
}

// Create creates a network resource (please check docs)
// name: is the name of the network resource. The Create function is idempotent which means if the same name is used the function
// will not recreate the resource.
// master: Normally the br-ndmz bridge, this is the resource "way out" to the public internet. A `public` interface is created and wired
// to the master bridge
// ndmzIP: the IP assigned to the `public` interface.
// ndmzGwIP: the gw Ip for the resource. Normally this is the ip assigned to the master bridge.
// privateNet: optional private network range
// seed: mycelium seed
func Create(name string, master *netlink.Bridge, ndmzIP *net.IPNet, ndmzGwIP *net.IPNet, privateNet *net.IPNet, seed []byte) (*Resource, error) {
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
				return nil, fmt.Errorf("couldn't create bridge %s: %w", name, err)
			}
		}
	}

	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		netNS, err = namespace.Create(nsName)
		if err != nil {
			return nil, fmt.Errorf("failed to create namespace %s", err)
		}
	}

	defer netNS.Close()

	if privateNet != nil {
		if privateNet.IP.To4() == nil {
			return nil, fmt.Errorf("private ip range must be IPv4 address")
		}

		if !ifaceutil.Exists(infPrivate, netNS) {
			privateLink, err := ifaceutil.MakeVethPair(infPrivate, privateNetBr, 1500, nsName[0:3])
			if err != nil {
				return nil, fmt.Errorf("failed to create private link: %w", err)
			}
			err = netlink.LinkSetNsFd(privateLink, int(netNS.Fd()))
			if err != nil {
				return nil, fmt.Errorf("failed to move public link %s to namespace:%s : %w", infPublic, netNS.Path(), err)
			}

		}
	}

	// create public interface and attach it to ndmz bridge
	if !ifaceutil.Exists(infPublic, netNS) {
		pubLink, err := ifaceutil.MakeVethPair(infPublic, master.Name, 1500, nsName[0:3])
		if err != nil {
			return nil, fmt.Errorf("failed to create public link in namespace %s: %w", nsName, err)
		}
		err = netlink.LinkSetNsFd(pubLink, int(netNS.Fd()))
		if err != nil {
			return nil, fmt.Errorf("failed to move public link %s to namespace:%s : %w", infPublic, netNS.Path(), err)
		}

	}

	if !ifaceutil.Exists(infMycelium, netNS) {
		myceliumLink, err := ifaceutil.MakeVethPair(infMycelium, myBr, 1500, nsName[0:3])
		if err != nil {
			return nil, fmt.Errorf("failed to create mycelium link: %w", err)
		}
		err = netlink.LinkSetNsFd(myceliumLink, int(netNS.Fd()))
		if err != nil {
			return nil, fmt.Errorf("failed to move mycelium link: %s to namespace:%s : %w", infMycelium, netNS.Path(), err)
		}
	}

	err = netNS.Do(func(_ ns.NetNS) error {
		if err := ifaceutil.SetLoUp(); err != nil {
			return fmt.Errorf("failed to set lo up for namespace '%s': %w", nsName, err)
		}

		if err := setLinkAddr(infPublic, ndmzIP); err != nil {
			return fmt.Errorf("couldn't set link addr for public interface in namespace %s: %w", nsName, err)
		}
		if err := options.SetIPv6Forwarding(true); err != nil {
			return fmt.Errorf("failed to enable ipv6 forwarding in namespace %q: %w", nsName, err)
		}

		if privateNet != nil {
			privGwIp := privateNet.IP.To4()
			// this is to take the first IP of the private range
			// and use it as the gw address for the entire private range
			privGwIp[net.IPv4len-1] = 1
			// this IP is then set on the private interface
			privateNet.IP = privGwIp
			if err := setLinkAddr(infPrivate, privateNet); err != nil {
				return fmt.Errorf("couldn't set link addr for private interface in namespace %s: %w", nsName, err)
			}
		}

		if err := netlink.RouteAdd(&netlink.Route{
			Gw: ndmzGwIP.IP,
		}); err != nil && !os.IsExist(err) {
			return fmt.Errorf("failed to add ndmz routing")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	rules, err := nftRules.Open("nft/rules.nft")
	if err != nil {
		return nil, fmt.Errorf("failed to load nft.rules file")
	}

	if err := nft.Apply(rules, nsName); err != nil {
		return nil, fmt.Errorf("failed to apply nft rules for namespace '%s': %w", name, err)
	}
	rules.Close()
	return &Resource{name}, setupMycelium(netNS, infMycelium, seed)
}

func Delete(name string) error {
	nsName := fmt.Sprintf("n%s", name)
	netNS, err := namespace.GetByName(nsName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	if err := destroyMycelium(netNS, zinit.Default()); err != nil {
		return err
	}

	if err := namespace.Delete(netNS); err != nil {
		return err
	}

	privateNetBr := fmt.Sprintf("r%s", name)
	myBr := fmt.Sprintf("m%s", name)

	if err := bridge.Delete(privateNetBr); err != nil {
		return err
	}

	return bridge.Delete(myBr)

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

// Get return resource handler
func Get(name string) (*Resource, error) {
	nsName := fmt.Sprintf("n%s", name)

	if namespace.Exists(nsName) {
		return &Resource{name}, nil
	}

	return nil, fmt.Errorf("resource not found: %s", name)
}

var networkResourceNet = net.IPNet{
	IP:   net.ParseIP("100.64.0.0"),
	Mask: net.IPv4Mask(0xff, 0xff, 0, 0),
}

func (r *Resource) AttachPrivate(id string, vmIp net.IP) (device pkg.TapDevice, err error) {
	nsName := fmt.Sprintf("n%s", r.name)
	netNs, err := namespace.GetByName(nsName)
	if err != nil {
		return
	}
	if vmIp[3] == 1 {
		return device, fmt.Errorf("ip %s is reserved", vmIp.String())
	}

	var addrs []netlink.Addr

	if err = netNs.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(infPrivate)
		if err != nil {
			return err
		}
		addrs, err = netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return
	}
	if len(addrs) != 1 {
		return device, fmt.Errorf("expect addresses on private interface to be 1 got %d", len(addrs))
	}
	gw := addrs[0].IPNet
	if !gw.Contains(vmIp) {
		return device, fmt.Errorf("ip not in range")
	}

	deviceName := ifaceutil.DeviceNameFromInputBytes([]byte(id))
	tapName := fmt.Sprintf("b-%s", deviceName)

	privateNetBr := fmt.Sprintf("r%s", r.name)
	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(tapName))

	ip := &net.IPNet{
		IP:   vmIp,
		Mask: gw.Mask,
	}
	_, getLinkErr := netlink.LinkByName(tapName)
	if getLinkErr != nil {
		mtap, err := tuntap.CreateTap(tapName, privateNetBr)
		if err != nil {
			return pkg.TapDevice{}, err
		}

		if err = netlink.AddrAdd(mtap, &netlink.Addr{
			IPNet: ip,
		}); err != nil {
			return pkg.TapDevice{}, err
		}
	}

	routes := []pkg.Route{
		{
			Gateway: gw.IP,
		},
		{
			Net:     networkResourceNet,
			Gateway: gw.IP,
		},
	}
	return pkg.TapDevice{
		Name:   tapName,
		Mac:    hw,
		IP:     ip,
		Routes: routes,
	}, nil
}

func (r *Resource) AttachMycelium(id string, seed []byte) (device pkg.TapDevice, err error) {
	nsName := fmt.Sprintf("n%s", r.name)
	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return
	}
	name := filepath.Base(netNS.Path())
	netSeed, err := os.ReadFile(filepath.Join(myceliumSeedDir, name))
	if err != nil {
		return
	}

	inspect, err := inspectMycelium(netSeed)
	if err != nil {
		return
	}

	ip, gw, err := inspect.IPFor(seed)
	if err != nil {
		return
	}

	deviceName := ifaceutil.DeviceNameFromInputBytes([]byte(id))
	tapName := fmt.Sprintf("m-%s", deviceName)

	myBr := fmt.Sprintf("m%s", r.name)
	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(tapName))

	_, err = tuntap.CreateTap(tapName, myBr)
	if err != nil {
		return
	}

	route := pkg.Route{

		Net: net.IPNet{
			IP:   net.ParseIP("400::"),
			Mask: net.CIDRMask(7, 128),
		},
		Gateway: gw.IP,
	}
	return pkg.TapDevice{
		Name:   tapName,
		IP:     &ip,
		Routes: []pkg.Route{route},
		Mac:    hw,
	}, nil
}

func (r *Resource) AttachMyceliumZDB(id string, zdbNS ns.NetNS) (err error) {
	nsName := fmt.Sprintf("n%s", r.name)
	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return
	}
	name := filepath.Base(netNS.Path())
	netSeed, err := os.ReadFile(filepath.Join(myceliumSeedDir, name))
	if err != nil {
		return
	}

	inspect, err := inspectMycelium(netSeed)
	if err != nil {
		return
	}
	seed := make([]byte, 6)

	_, err = rand.Read(seed)
	if err != nil {
		return
	}

	ip, gw, err := inspect.IPFor(seed)
	if err != nil {
		return
	}

	deviceName := ifaceutil.DeviceNameFromInputBytes([]byte(id))
	linkName := fmt.Sprintf("m-%s", deviceName)

	//
	if !ifaceutil.Exists(linkName, zdbNS) {
		zdbLink, err := ifaceutil.MakeVethPair(linkName, "mdmz", 1500, nsName[0:3])
		if err != nil {
			return fmt.Errorf("failed to create zdb link %s : %w", linkName, err)
		}
		err = netlink.LinkSetNsFd(zdbLink, int(zdbNS.Fd()))
		if err != nil {
			return fmt.Errorf("failed to move zdb link: %s to namespace:%s : %w", linkName, netNS.Path(), err)
		}

		return zdbNS.Do(func(_ ns.NetNS) error {
			err = setLinkAddr(linkName, &ip)
			if err != nil {
				return err
			}

			if err := ifaceutil.SetLoUp(); err != nil {
				return fmt.Errorf("failed to set lo up for namespace '%s': %w", nsName, err)
			}

			if err := options.SetIPv6Forwarding(true); err != nil {
				return fmt.Errorf("failed to enable ipv6 forwarding in namespace %q: %w", nsName, err)
			}

			return netlink.RouteAdd(&netlink.Route{
				Dst: &net.IPNet{
					IP:   net.ParseIP("400::"),
					Mask: net.CIDRMask(7, 128),
				},
				Gw: gw.IP,
			})

		})
	}
	return nil
	//

}

func (r *Resource) Seed() (seed []byte, err error) {
	nsName := fmt.Sprintf("n%s", r.name)
	netNS, err := namespace.GetByName(nsName)
	if err != nil {
		return
	}
	name := filepath.Base(netNS.Path())
	seed, err = os.ReadFile(filepath.Join(myceliumSeedDir, name))
	return
}
