package netlight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/blang/semver"
	"github.com/cenkalti/backoff"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos4/pkg"
	"github.com/threefoldtech/zos4/pkg/cache"
	"github.com/threefoldtech/zos4/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos4/pkg/netlight/bridge"
	"github.com/threefoldtech/zos4/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zos4/pkg/netlight/ipam"
	"github.com/threefoldtech/zos4/pkg/netlight/namespace"
	"github.com/threefoldtech/zos4/pkg/netlight/options"
	"github.com/threefoldtech/zos4/pkg/netlight/resource"
	"github.com/threefoldtech/zos4/pkg/versioned"
	"github.com/vishvananda/netlink"
)

const (
	NDMZBridge    = "br-ndmz"
	NDMZGw        = "gw"
	mib           = 1024 * 1024
	ipamLeaseDir  = "ndmz-lease"
	DefaultBridge = "zos"
	networkDir    = "networks"
)

var (
	NDMZGwIP = &net.IPNet{
		IP:   net.ParseIP("100.127.0.1"),
		Mask: net.CIDRMask(16, 32),
	}
)

var NetworkSchemaLatestVersion = semver.MustParse("0.1.0")

type networker struct {
	ipamLease  string
	networkDir string
}

var _ pkg.NetworkerLight = (*networker)(nil)

func NewNetworker() (pkg.NetworkerLight, error) {
	vd, err := cache.VolatileDir("networkd", 50*mib)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create networkd cache directory: %w", err)
	}

	ipamLease := filepath.Join(vd, ipamLeaseDir)
	runtimeDir := filepath.Join(vd, networkDir)

	return &networker{
		ipamLease:  ipamLease,
		networkDir: runtimeDir,
	}, nil
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

func (n *networker) AttachZDB(id string) (string, error) {
	name := ifaceutil.DeviceNameFromInputBytes([]byte(id))
	nsName := fmt.Sprintf("n%s", name)

	ns, err := namespace.GetByName(nsName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if ns == nil {
		ns, err = namespace.Create(nsName)
		if err != nil {
			return "", err
		}
	}

	r, err := resource.Get("dmz")
	if err != nil {
		return "", err
	}

	return nsName, r.AttachMyceliumZDB(id, ns)
}

// GetSubnet of a local network resource identified by the network ID, ipv4 and ipv6
// subnet respectively
func (n *networker) GetSubnet(networkID pkg.NetID) (net.IPNet, error) {
	localNR, err := n.networkOf(networkID)
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	return localNR.Subnet.IPNet, nil
}

func (n *networker) networkOf(id zos.NetID) (nr pkg.Network, err error) {
	path := filepath.Join(n.networkDir, string(id))
	file, err := os.OpenFile(path, os.O_RDWR, 0660)
	if err != nil {
		return nr, err
	}
	defer file.Close()

	reader, err := versioned.NewReader(file)
	if versioned.IsNotVersioned(err) {
		// old data that doesn't have any version information
		if _, err := file.Seek(0, 0); err != nil {
			return nr, err
		}

		reader = versioned.NewVersionedReader(NetworkSchemaLatestVersion, file)
	} else if err != nil {
		return nr, err
	}

	var net pkg.Network
	dec := json.NewDecoder(reader)

	version := reader.Version()
	// validV1 := versioned.MustParseRange(fmt.Sprintf("=%s", pkg.NetworkSchemaV1))
	validLatest := versioned.MustParseRange(fmt.Sprintf("<=%s", NetworkSchemaLatestVersion.String()))

	if validLatest(version) {
		if err := dec.Decode(&net); err != nil {
			return nr, err
		}
	} else {
		return nr, fmt.Errorf("unknown network object version (%s)", version)
	}

	return net, nil
}

func (n *networker) ZDBIPs(zdbNamespace string) ([]net.IP, error) {
	ips := make([]net.IP, 0)

	netNs, err := namespace.GetByName(zdbNamespace)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	err = netNs.Do(func(_ ns.NetNS) error {
		links, err := netlink.LinkList()
		if err != nil {
			return err
		}
		for _, link := range links {
			addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
			if err != nil {
				return err
			}
			for _, addr := range addrs {
				ips = append(ips, addr.IP)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ips, nil
}

func (n *networker) Ready() error {
	return nil
}

func (n *networker) Namespace(id string) string {
	return fmt.Sprintf("n%s", id)
}

func (n *networker) ZOSAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	var index int
	_ = backoff.Retry(func() error {
		link, err := netlink.LinkByName(DefaultBridge)
		if err != nil {
			log.Error().Err(err).Msg("can't get defaut bridge")
			return err
		}
		index = link.Attrs().Index
		return nil
	}, backoff.NewConstantBackOff(2*time.Second))

	get := func() pkg.NetlinkAddresses {
		var result pkg.NetlinkAddresses
		link, err := netlink.LinkByName(DefaultBridge)
		if err != nil {
			log.Error().Err(err).Msgf("could not find the '%s' bridge", DefaultBridge)
			return nil
		}
		values, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			log.Error().Err(err).Msgf("could not list the '%s' bridge ips", DefaultBridge)
			return nil
		}
		for _, value := range values {
			result = append(result, *value.IPNet)
		}

		slices.SortFunc(result, func(a, b net.IPNet) int {
			return bytes.Compare(a.IP, b.IP)
		})

		return result
	}

	updateChan := make(chan netlink.AddrUpdate)
	if err := netlink.AddrSubscribe(updateChan, ctx.Done()); err != nil {
		log.Error().Err(err).Msgf("could not subscribe to addresses updates")
		return nil
	}

	ch := make(chan pkg.NetlinkAddresses)
	var current pkg.NetlinkAddresses
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case update := <-updateChan:
				if update.LinkIndex != index || !update.NewAddr {
					continue
				}
				new := get()
				if slices.CompareFunc(current, new, func(a, b net.IPNet) int {
					return bytes.Compare(a.IP, b.IP)
				}) == 0 {
					// if the 2 sets of IPs are identitcal, we don't
					// trigger the event
					continue
				}
				current = new
				ch <- new
			}
		}
	}()

	return ch
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
	err = netlink.AddrAdd(link, &netlink.Addr{IPNet: NDMZGwIP})
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return nil, err
	}

	return link.(*netlink.Bridge), nil
}
