package netlight

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/cache"
	"github.com/threefoldtech/zos/pkg/netlight/bridge"
	"github.com/threefoldtech/zos/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zos/pkg/netlight/ipam"
	"github.com/threefoldtech/zos/pkg/netlight/macvlan"
	"github.com/threefoldtech/zos/pkg/netlight/namespace"
	"github.com/threefoldtech/zos/pkg/netlight/options"
	"github.com/threefoldtech/zos/pkg/netlight/resource"
	"github.com/vishvananda/netlink"
)

const (
	NDMZBridge   = "br-ndmz"
	NDMZGw       = "gw"
	mib          = 1024 * 1024
	ipamLeaseDir = "ndmz-lease"
)

var (
	NDMZGwIP = &net.IPNet{
		IP:   net.ParseIP("100.127.0.1"),
		Mask: net.CIDRMask(16, 32),
	}
)

type networker struct {
	ipamLease string
}

var _ pkg.NetworkerLight = (*networker)(nil)

func NewNetworker() (pkg.NetworkerLight, error) {
	vd, err := cache.VolatileDir("networkd", 50*mib)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create networkd cache directory: %w", err)
	}

	ipamLease := filepath.Join(vd, ipamLeaseDir)
	return &networker{ipamLease: ipamLease}, nil
}

func (n *networker) Create(name string, privateNet net.IPNet, seed []byte) error {
	b, err := bridge.Get(NDMZBridge)
	if err != nil {
		return err
	}
	ip, err := ipam.AllocateIPv4(name, n.ipamLease)
	if err != nil {
		return err
	}

	_, err = resource.Create(name, b, ip, NDMZGwIP, &privateNet, seed)
	return err
}

func (n *networker) Delete(name string) error {
	if err := ipam.DeAllocateIPv4(name, n.ipamLease); err != nil {
		return err
	}

	return resource.Delete(name)

}

func (n *networker) AttachPrivate(name, id string, vmIp net.IP) (device pkg.TapDevice, err error) {
	resource, err := resource.Get(name)
	if err != nil {
		return
	}
	return resource.AttachPrivate(id, vmIp)
}

func (n *networker) AttachMycelium(name, id string, seed []byte) (device pkg.TapDevice, err error) {
	resource, err := resource.Get(name)
	if err != nil {
		return
	}
	return resource.AttachMycelium(id, seed)
}

// detach everything for this id
func (n *networker) Detach(id string) error {
	// delete all tap devices for both mycelium and priv (if exists)
	deviceName := ifaceutil.DeviceNameFromInputBytes([]byte(id))
	myName := fmt.Sprintf("m-%s", deviceName)

	if err := ifaceutil.Delete(myName, nil); err != nil {
		return err
	}

	tapName := fmt.Sprintf("b-%s", deviceName)

	return ifaceutil.Delete(tapName, nil)
}

func (n *networker) Interfaces(iface string, netns string) (pkg.Interfaces, error) {
	getter := func(iface string) ([]netlink.Link, error) {
		if iface != "" {
			l, err := netlink.LinkByName(iface)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get interface %s", iface)
			}
			return []netlink.Link{l}, nil
		}

		all, err := netlink.LinkList()
		if err != nil {
			return nil, err
		}
		filtered := all[:0]
		for _, l := range all {
			name := l.Attrs().Name

			if name == "lo" ||
				(l.Type() != "device" && name != "zos") {

				continue
			}

			filtered = append(filtered, l)
		}

		return filtered, nil
	}

	interfaces := make(map[string]pkg.Interface)
	f := func(_ ns.NetNS) error {
		links, err := getter(iface)
		if err != nil {
			return errors.Wrapf(err, "failed to get interfaces (query: '%s')", iface)
		}

		for _, link := range links {

			addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
			if err != nil {
				return errors.Wrapf(err, "failed to list addresses of interfaces %s", iface)
			}
			ips := make([]net.IPNet, 0, len(addrs))
			for _, addr := range addrs {
				ip := addr.IP
				if ip6 := ip.To16(); ip6 != nil {
					// ipv6
					if !ip6.IsGlobalUnicast() || ifaceutil.IsULA(ip6) {
						// skip if not global or is ula address
						continue
					}
				}

				ips = append(ips, *addr.IPNet)
			}

			interfaces[link.Attrs().Name] = pkg.Interface{
				Name: link.Attrs().Name,
				Mac:  link.Attrs().HardwareAddr.String(),
				IPs:  ips,
			}
		}

		return nil
	}

	if netns != "" {
		netNS, err := namespace.GetByName(netns)
		if err != nil {
			return pkg.Interfaces{}, errors.Wrapf(err, "failed to get network namespace %s", netns)
		}
		defer netNS.Close()

		if err := netNS.Do(f); err != nil {
			return pkg.Interfaces{}, err
		}
	} else {
		if err := f(nil); err != nil {
			return pkg.Interfaces{}, err
		}
	}

	return pkg.Interfaces{Interfaces: interfaces}, nil
}

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

		err = netlink.AddrAdd(gwLink, &netlink.Addr{IPNet: NDMZGwIP})
		if err != nil && !os.IsExist(err) {
			return nil, err
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
