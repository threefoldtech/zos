package resource

import (
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
	"github.com/threefoldtech/zos/pkg/netlight/macvlan"
	"github.com/threefoldtech/zos/pkg/netlight/macvtap"
	"github.com/threefoldtech/zos/pkg/netlight/namespace"
	"github.com/threefoldtech/zos/pkg/netlight/options"
	"github.com/threefoldtech/zos/pkg/network/nft"
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
			if _, err = macvlan.Create(infPrivate, privateNetBr, netNS); err != nil {
				return nil, fmt.Errorf("failed to create private link: %w", err)
			}
		}
	}

	// create public interface and attach it to ndmz bridge
	if !ifaceutil.Exists(infPublic, netNS) {
		if _, err = macvlan.Create(infPublic, master.Name, netNS); err != nil {
			return nil, fmt.Errorf("failed to create public link: %w", err)
		}
	}

	if !ifaceutil.Exists(infMycelium, netNS) {
		if _, err = macvlan.Create(infMycelium, myBr, netNS); err != nil {
			return nil, err
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

		// if err := setLinkAddr(infPrivate, )
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

// myceliumIpSeed is 6 bytes
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

	mtap, err := macvtap.CreateMACvTap(tapName, privateNetBr, hw)
	if err != nil {
		return
	}
	ip := &net.IPNet{
		IP:   vmIp,
		Mask: gw.Mask,
	}

	if err = netlink.AddrAdd(mtap, &netlink.Addr{
		IPNet: ip,
	}); err != nil {
		return
	}

	return pkg.TapDevice{
		Name:    tapName,
		Mac:     hw,
		IP:      ip,
		Gateway: gw,
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

	_, err = macvtap.CreateMACvTap(tapName, myBr, hw)
	if err != nil {
		return
	}

	// if err = netlink.AddrAdd(mtap, &netlink.Addr{
	// 	IPNet: &ip,
	// }); err != nil {
	// 	return
	// }

	return pkg.TapDevice{
		Name:    tapName,
		IP:      &ip,
		Gateway: &gw,
		Mac:     hw,
	}, nil
}
