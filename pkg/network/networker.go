package network

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/termie/go-shutil"

	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg/cache"
	"github.com/threefoldtech/zos/pkg/network/macvtap"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/public"
	"github.com/threefoldtech/zos/pkg/network/tuntap"
	"github.com/threefoldtech/zos/pkg/network/wireguard"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"

	"github.com/vishvananda/netlink"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zos/pkg/network/ifaceutil"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/nr"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/set"
	"github.com/threefoldtech/zos/pkg/versioned"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/network/namespace"

	"github.com/threefoldtech/zos/pkg"
)

const (
	// ZDBIface is the name of the interface used in the 0-db network namespace
	ZDBIface           = "zdb0"
	networkDir         = "networks"
	ipamLeaseDir       = "ndmz-lease"
	ipamPath           = "/var/cache/modules/networkd/lease"
	zdbNamespacePrefix = "zdb-ns-"
)

const (
	// NodeExporterPort is reserved for node exporter
	NodeExporterPort = 9100
	mib              = 1024 * 1024
)

type networker struct {
	identity     pkg.IdentityManager
	networkDir   string
	ipamLeaseDir string
	tnodb        client.Directory
	portSet      *set.UIntSet

	ndmz ndmz.DMZ
	ygg  *yggdrasil.Server
}

// NewNetworker create a new pkg.Networker that can be used over zbus
func NewNetworker(identity pkg.IdentityManager, tnodb client.Directory, storageDir string, ndmz ndmz.DMZ, ygg *yggdrasil.Server) (pkg.Networker, error) {
	vd, err := cache.VolatileDir("networkd", 50*mib)
	if err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("failed to create networkd cache directory: %w", err)
	}

	nwDir := filepath.Join(vd, networkDir)
	ipamLease := filepath.Join(vd, ipamLeaseDir)

	oldPath := filepath.Join(ipamPath, "ndmz")
	newPath := filepath.Join(ipamLease, "ndmz")
	if _, err := os.Stat(oldPath); err == nil {
		if err := shutil.CopyTree(oldPath, newPath, nil); err != nil {
			return nil, err
		}
		_ = os.RemoveAll(ipamPath)
	}

	//TODO: remove once all the node have move the network into the volatile directory
	if _, err = os.Stat(storageDir); err == nil {
		if err := copyNetworksToVolatile(storageDir, nwDir); err != nil {
			return nil, fmt.Errorf("failed to copy old networks directory: %w", err)
		}
	}

	nw := &networker{
		identity:     identity,
		tnodb:        tnodb,
		networkDir:   nwDir,
		ipamLeaseDir: ipamLease,
		portSet:      set.NewInt(),

		ygg:  ygg,
		ndmz: ndmz,
	}

	// always add the reserved yggdrasil port to the port set so we make sure they are never
	// picked for wireguard endpoints
	for _, port := range []int{yggdrasil.YggListenTCP, yggdrasil.YggListenTLS, yggdrasil.YggListenLinkLocal, NodeExporterPort} {
		if err := nw.portSet.Add(uint(port)); err != nil && errors.Is(err, set.ErrConflict{}) {
			return nil, err
		}
	}

	if err := nw.syncWGPorts(); err != nil {
		return nil, err
	}

	if err := nw.publishWGPorts(); err != nil {
		return nil, err
	}

	return nw, nil
}

func copyNetworksToVolatile(src, dst string) error {
	log.Info().Msg("move network cached file to volatile directory")
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	copy := func(src string, dst string) error {
		log.Info().Str("source", src).Str("dst", dst).Msg("copy file")
		// Read all content of src to data
		data, err := ioutil.ReadFile(src)
		if err != nil {
			return err
		}
		// Write data to dst
		return ioutil.WriteFile(dst, data, 0644)
	}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		s := filepath.Join(src, info.Name())
		d := filepath.Join(dst, info.Name())
		if err := copy(s, d); err != nil {
			return err
		}
	}
	return nil
}

var _ pkg.Networker = (*networker)(nil)

func (n *networker) Ready() error {
	return nil
}

func (n *networker) Join(networkdID pkg.NetID, containerID string, cfg pkg.ContainerNetworkConfig) (join pkg.Member, err error) {
	// TODO:
	// 1- Make sure this network id is actually deployed
	// 2- Check if the requested network config is doable
	// 3- Create a new namespace, then create a veth pair inside this namespace
	// 4- Hook one end to the NR bridge
	// 5- Assign IP to the veth endpoint inside the namespace.
	// 6- return the namespace name

	log.Info().Str("network-id", string(networkdID)).Msg("joining network")

	localNR, err := n.networkOf(string(networkdID))
	if err != nil {
		return join, errors.Wrapf(err, "couldn't load network with id (%s)", networkdID)
	}

	ipv4Only, err := n.ndmz.IsIPv4Only()
	if err != nil {
		return join, errors.Wrap(err, "failed to check ipv6 support")
	}
	if cfg.PublicIP6 && ipv4Only {
		return join, errors.Errorf("this node runs in IPv4 only mode and you asked for a public IPv6. Impossible to fulfill the request")
	}

	netRes, err := nr.New(localNR)
	if err != nil {
		return join, errors.Wrap(err, "failed to load network resource")
	}

	ips := make([]net.IP, len(cfg.IPs))
	for i, addr := range cfg.IPs {
		ips[i] = net.ParseIP(addr)
	}

	join, err = netRes.Join(nr.ContainerConfig{
		ContainerID: containerID,
		IPs:         ips,
		PublicIP6:   cfg.PublicIP6,
		IPv4Only:    ipv4Only,
	})
	if err != nil {
		return join, errors.Wrap(err, "failed to load network resource")
	}

	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(containerID))
	netNs, err := namespace.GetByName(join.Namespace)
	if err != nil {
		return join, errors.Wrap(err, "failed to found a valid network interface to use as parent for 0-db container")
	}
	defer netNs.Close()

	if cfg.PublicIP6 {
		if err = n.createMacVlan("pub", hw, nil, nil, netNs); err != nil {
			return join, errors.Wrap(err, "failed to create public macvlan interface")
		}
	}

	if cfg.YggdrasilIP {
		var (
			ips    []*net.IPNet
			routes []*netlink.Route
		)

		ip, err := n.ygg.SubnetFor(hw)
		if err != nil {
			return join, err
		}

		ips = []*net.IPNet{
			{
				IP:   ip,
				Mask: net.CIDRMask(64, 128),
			},
		}
		join.YggdrasilIP = ip

		gw, err := n.ygg.Gateway()
		if err != nil {
			return join, err
		}

		routes = []*netlink.Route{
			{
				Dst: &net.IPNet{
					IP:   net.ParseIP("200::"),
					Mask: net.CIDRMask(7, 128),
				},
				Gw: gw.IP,
				// LinkIndex:... this is set by macvlan.Install
			},
		}

		if err := n.createMacVlan("ygg", hw, ips, routes, netNs); err != nil {
			return join, errors.Wrap(err, "failed to create yggdrasil macvlan interface")
		}
	}

	return join, nil
}

func (n *networker) Leave(networkdID pkg.NetID, containerID string) error {
	log.Info().Str("network-id", string(networkdID)).Msg("leaving network")

	localNR, err := n.networkOf(string(networkdID))
	if err != nil {
		return errors.Wrapf(err, "couldn't load network with id (%s)", networkdID)
	}

	netRes, err := nr.New(localNR)
	if err != nil {
		return errors.Wrap(err, "failed to load network resource")
	}

	return netRes.Leave(containerID)
}

// ZDBPrepare sends a macvlan interface into the
// network namespace of a ZDB container
func (n networker) ZDBPrepare(hw net.HardwareAddr) (string, error) {
	netNSName := zdbNamespacePrefix + strings.Replace(hw.String(), ":", "", -1)

	netNs, err := createNetNS(netNSName)
	if err != nil {
		return "", err
	}
	defer netNs.Close()

	var (
		ips    []*net.IPNet
		routes []*netlink.Route
	)

	if n.ygg != nil {
		ip, err := n.ygg.SubnetFor(hw)
		if err != nil {
			return "", fmt.Errorf("failed to generate ygg subnet IP: %w", err)
		}

		ips = []*net.IPNet{
			{
				IP:   ip,
				Mask: net.CIDRMask(64, 128),
			},
		}

		gw, err := n.ygg.Gateway()
		if err != nil {
			return "", fmt.Errorf("failed to get ygg gateway IP: %w", err)
		}

		routes = []*netlink.Route{
			{
				Dst: &net.IPNet{
					IP:   net.ParseIP("200::"),
					Mask: net.CIDRMask(7, 128),
				},
				Gw: gw.IP,
				// LinkIndex:... this is set by macvlan.Install
			},
		}

	}

	return netNSName, n.createMacVlan(ZDBIface, hw, ips, routes, netNs)
}

// ZDBDestroy is the opposite of ZDPrepare, it makes sure network setup done
// for zdb is rewind. ns param is the namespace return by the ZDBPrepare
func (n networker) ZDBDestroy(ns string) error {
	if !strings.HasPrefix(ns, zdbNamespacePrefix) {
		return fmt.Errorf("invalid zdb namespace name '%s'", ns)
	}

	if !namespace.Exists(ns) {
		return nil
	}

	nSpace, err := namespace.GetByName(ns)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to get namespace '%s'", ns)
	}

	return namespace.Delete(nSpace)
}

func (n networker) createMacVlan(iface string, hw net.HardwareAddr, ips []*net.IPNet, routes []*netlink.Route, netNs ns.NetNS) error {
	var macVlan *netlink.Macvlan
	err := netNs.Do(func(_ ns.NetNS) error {
		var err error
		macVlan, err = macvlan.GetByName(iface)
		return err
	})

	if _, ok := err.(netlink.LinkNotFoundError); ok {
		macVlan, err = macvlan.Create(iface, public.PublicBridge, netNs)

		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	log.Debug().Str("HW", hw.String()).Str("macvlan", macVlan.Name).Msg("setting hw address on link")
	// we don't set any route or ip
	if err := macvlan.Install(macVlan, hw, ips, routes, netNs); err != nil {
		return err
	}

	return nil
}

// SetupTap interface in the network resource. We only allow 1 tap interface to be
// set up per NR currently
func (n *networker) SetupTap(networkID pkg.NetID) (string, error) {
	log.Info().Str("network-id", string(networkID)).Msg("Setting up tap interface")

	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return "", errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	netRes, err := nr.New(localNR)
	if err != nil {
		return "", errors.Wrap(err, "failed to load network resource")
	}

	bridgeName, err := netRes.BridgeName()
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace bridge")
	}

	tapIface, err := tapName(networkID)
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace tap device name")
	}

	_, err = tuntap.CreateTap(tapIface, bridgeName)

	return tapIface, err
}

func (n *networker) TapExists(networkID pkg.NetID) (bool, error) {
	log.Info().Str("network-id", string(networkID)).Msg("Checking if tap interface exists")

	tapIface, err := tapName(networkID)
	if err != nil {
		return false, errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Exists(tapIface, nil), nil
}

// RemoveTap in the network resource.
func (n *networker) RemoveTap(networkID pkg.NetID) error {
	log.Info().Str("network-id", string(networkID)).Msg("Removing tap interface")

	tapIface, err := tapName(networkID)
	if err != nil {
		return errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Delete(tapIface, nil)
}

func (n *networker) PublicIPv4Support() bool {
	return n.ndmz.SupportsPubIPv4()
}

// SetupPubTap sets up a tap device in the host namespace for the public ip
// reservation id. It is hooked to the public bridge. The name of the tap
// interface is returned
func (n *networker) SetupPubTap(pubIPReservationID string) (string, error) {
	log.Info().Str("pubip-res-id", string(pubIPReservationID)).Msg("Setting up public tap interface")

	if !n.ndmz.SupportsPubIPv4() {
		return "", errors.New("can't create public tap on this node")
	}

	tapIface, err := pubTapName(pubIPReservationID)
	if err != nil {
		return "", errors.Wrap(err, "could not get network namespace tap device name")
	}

	hw := ifaceutil.HardwareAddrFromInputBytes([]byte(pubIPReservationID))
	_, err = macvtap.CreateMACvTap(tapIface, public.PublicBridge, hw)

	return tapIface, err
}

// PubTapExists checks if the tap device for the public network exists already
func (n *networker) PubTapExists(pubIPReservationID string) (bool, error) {
	log.Info().Str("pubip-res-id", string(pubIPReservationID)).Msg("Checking if public tap interface exists")

	tapIface, err := pubTapName(pubIPReservationID)
	if err != nil {
		return false, errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Exists(tapIface, nil), nil
}

// RemovePubTap removes the public tap device from the host namespace
// of the networkID
func (n *networker) RemovePubTap(pubIPReservationID string) error {
	log.Info().Str("pubip-res-id", string(pubIPReservationID)).Msg("Removing public tap interface")

	tapIface, err := pubTapName(pubIPReservationID)
	if err != nil {
		return errors.Wrap(err, "could not get network namespace tap device name")
	}

	return ifaceutil.Delete(tapIface, nil)
}

// GetPublicIPv6Subnet returns the IPv6 prefix op the public subnet of the host
func (n *networker) GetPublicIPv6Subnet() (net.IPNet, error) {
	addrs, err := n.ndmz.GetIP(ndmz.FamilyV6)
	if err != nil {
		return net.IPNet{}, errors.Wrap(err, "could not get ips from ndmz")
	}

	for _, addr := range addrs {
		if addr.IP.IsGlobalUnicast() && !isULA(addr.IP) && !isYgg(addr.IP) {
			return addr, nil
		}
	}

	return net.IPNet{}, fmt.Errorf("no public ipv6 found")
}

// GetSubnet of a local network resource identified by the network ID, ipv4 and ipv6
// subnet respectively
func (n networker) GetSubnet(networkID pkg.NetID) (net.IPNet, error) {
	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	return localNR.Subnet.IPNet, nil
}

// GetNet of a network identified by the network ID
func (n networker) GetNet(networkID pkg.NetID) (net.IPNet, error) {
	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return net.IPNet{}, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	return localNR.NetworkIPRange.IPNet, nil
}

// GetDefaultGwIP returns the IPs of the default gateways inside the network
// resource identified by the network ID on the local node, for IPv4 and IPv6
// respectively
func (n networker) GetDefaultGwIP(networkID pkg.NetID) (net.IP, net.IP, error) {
	localNR, err := n.networkOf(string(networkID))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't load network with id (%s)", networkID)
	}

	// only IP4 atm
	ip := localNR.Subnet.IP.To4()
	if ip == nil {
		return nil, nil, errors.New("nr subnet is not valid IPv4")
	}

	// defaut gw is currently implied to be at `x.x.x.1`
	// also a subnet in a NR is assumed to be a /24
	ip[len(ip)-1] = 1

	// ipv6 is derived from the ipv4
	return ip, nr.Convert4to6(string(networkID), ip), nil
}

// GetIPv6From4 generates an IPv6 address from a given IPv4 address in a NR
func (n networker) GetIPv6From4(networkID pkg.NetID, ip net.IP) (net.IPNet, error) {
	if ip.To4() == nil {
		return net.IPNet{}, errors.New("invalid IPv4 address")
	}
	return net.IPNet{IP: nr.Convert4to6(string(networkID), ip), Mask: net.CIDRMask(64, 128)}, nil
}

// Addrs return the IP addresses of interface
func (n networker) Addrs(iface string, netns string) ([]net.IP, error) {
	var ips []net.IP

	f := func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(iface)
		if err != nil {
			return errors.Wrapf(err, "failed to get interface %s", iface)
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		if err != nil {
			return errors.Wrapf(err, "failed to list addresses of interfaces %s", iface)
		}
		ips = make([]net.IP, len(addrs))
		for i, addr := range addrs {
			ips[i] = addr.IP
		}
		return nil
	}

	if netns != "" {
		netNS, err := namespace.GetByName(netns)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get network namespace %s", netns)
		}
		defer netNS.Close()

		if err := netNS.Do(f); err != nil {
			return nil, err
		}
	} else {
		if err := f(nil); err != nil {
			return nil, err
		}
	}

	return ips, nil
}

// CreateNR implements pkg.Networker interface
func (n *networker) CreateNR(netNR pkg.NetResource) (string, error) {
	defer func() {
		if err := n.publishWGPorts(); err != nil {
			log.Warn().Err(err).Msg("failed to publish wireguard port to BCDB")
		}
	}()

	log.Info().Str("network", string(netNR.NetID)).Msg("create network resource")

	privateKey, err := n.extractPrivateKey(netNR.WGPrivateKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to extract private key from network object")
	}

	// check if there is a reserved wireguard port for this NR already
	// or if we need to update it
	storedNR, err := n.networkOf(string(netNR.NetID))
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err == nil {
		if err := n.releasePort(storedNR.WGListenPort); err != nil {
			return "", err
		}
	}

	if err := n.reservePort(netNR.WGListenPort); err != nil {
		return "", err
	}

	netr, err := nr.New(netNR)
	if err != nil {
		return "", err
	}

	cleanup := func() {
		log.Error().Msg("clean up network resource")
		if err := netr.Delete(); err != nil {
			log.Error().Err(err).Msg("error during deletion of network resource after failed deployment")
		}
		if err := n.releasePort(netNR.WGListenPort); err != nil {
			log.Error().Err(err).Msg("release wireguard port failed")
		}
	}

	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	wgName, err := netr.WGName()
	if err != nil {
		return "", errors.Wrap(err, "failed to get wg interface name for network resource")
	}

	log.Info().Msg("create network resource namespace")
	if err = netr.Create(); err != nil {
		return "", errors.Wrap(err, "failed to create network resource")
	}

	exists, err := netr.HasWireguard()
	if err != nil {
		return "", errors.Wrap(err, "failed to check if network resource has wireguard setup")
	}

	if !exists {
		var wg *wireguard.Wireguard
		wg, err = public.NewWireguard(wgName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create wg interface for network resource '%s'", netNR.Name)
		}
		if err = netr.SetWireguard(wg); err != nil {
			return "", errors.Wrap(err, "failed to setup wireguard interface for network resource")
		}
	}

	if err = n.ndmz.AttachNR(string(netNR.NetID), netr, n.ipamLeaseDir); err != nil {
		return "", errors.Wrapf(err, "failed to attach network resource to DMZ bridge")
	}

	if err = netr.ConfigureWG(privateKey); err != nil {
		return "", errors.Wrap(err, "failed to configure network resource")
	}

	if err = n.storeNetwork(netNR); err != nil {
		return "", errors.Wrap(err, "failed to store network object")
	}

	return netr.Namespace()
}

func (n *networker) storeNetwork(network pkg.NetResource) error {
	// map the network ID to the network namespace
	path := filepath.Join(n.networkDir, string(network.NetID))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := versioned.NewWriter(file, pkg.NetworkSchemaLatestVersion)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(writer)
	if err := enc.Encode(&network); err != nil {
		return err
	}

	return nil
}

// DeleteNR implements pkg.Networker interface
func (n *networker) DeleteNR(netNR pkg.NetResource) error {
	defer func() {
		if err := n.publishWGPorts(); err != nil {
			log.Warn().Msg("failed to publish wireguard port to BCDB")
		}
	}()

	nr, err := nr.New(netNR)
	if err != nil {
		return errors.Wrap(err, "failed to load network resource")
	}

	if err := nr.Delete(); err != nil {
		return errors.Wrap(err, "failed to delete network resource")
	}

	if err := n.releasePort(netNR.WGListenPort); err != nil {
		log.Error().Err(err).Msg("release wireguard port failed")
		// TODO: should we return the error ?
	}

	// map the network ID to the network namespace
	path := filepath.Join(n.networkDir, string(netNR.NetID))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("failed to remove file mapping between network ID and namespace")
	}

	return nil
}

func (n *networker) networkOf(id string) (nr pkg.NetResource, err error) {
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

		reader = versioned.NewVersionedReader(versioned.MustParse("0.0.0"), file)
	} else if err != nil {
		return nr, err
	}

	var net pkg.NetResource
	dec := json.NewDecoder(reader)

	version := reader.Version()
	validV1 := versioned.MustParseRange(fmt.Sprintf("=%s", pkg.NetworkSchemaV1))
	validLatest := versioned.MustParseRange(fmt.Sprintf("<=%s", pkg.NetworkSchemaLatestVersion))

	if validV1(version) {
		// we found a v1 full network definition, let migrate it to v2 network resource
		var network pkg.Network
		if err := dec.Decode(&network); err != nil {
			return nr, err
		}

		for _, nr := range network.NetResources {
			if nr.NodeID == n.identity.NodeID().Identity() {
				net = nr
				break
			}
		}
		net.Name = network.Name
		net.NetworkIPRange = network.IPRange
		net.NetID = network.NetID

		// overwrite the old version network with latest version
		// old data that doesn't have any version information
		if _, err := file.Seek(0, 0); err != nil {
			return nr, err
		}

		writer, err := versioned.NewWriter(file, pkg.NetworkSchemaLatestVersion)
		if err != nil {
			return nr, err
		}

		if err := json.NewEncoder(writer).Encode(&net); err != nil {
			return nr, err
		}

	} else if validLatest(version) {
		if err := dec.Decode(&net); err != nil {
			return nr, err
		}
	} else {
		return nr, fmt.Errorf("unknown network object version (%s)", version)
	}

	if err := net.Valid(); err != nil {
		return net, errors.Wrapf(err, "failed to validate cached network resource: %s", net.NetID)
	}

	return net, nil
}

func (n *networker) extractPrivateKey(hexKey string) (string, error) {
	//FIXME zaibon: I would like to move this into the nr package,
	// but this method requires the identity module which is only available
	// on the networker object
	sk, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", err
	}
	decoded, err := n.identity.Decrypt(sk)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func (n *networker) reservePort(port uint16) error {
	log.Debug().Uint16("port", port).Msg("reserve wireguard port")
	err := n.portSet.Add(uint(port))
	if err != nil {
		return errors.Wrap(err, "wireguard listen port already in use, pick another one")
	}

	return nil
}

func (n *networker) releasePort(port uint16) error {
	log.Debug().Uint16("port", port).Msg("release wireguard port")
	n.portSet.Remove(uint(port))
	return nil
}

func (n *networker) publishWGPorts() error {
	ports, err := n.portSet.List()
	if err != nil {
		return err
	}

	if err := n.tnodb.NodeSetPorts(n.identity.NodeID().Identity(), ports); err != nil {
		// maybe retry a couple of times ?
		// having bdb and the node out of sync is pretty bad
		return errors.Wrap(err, "fail to publish wireguard port to bcdb")
	}
	return nil
}

func (n *networker) DMZAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ips, err := n.ndmz.GetIP(ndmz.FamilyAll)
				if err != nil {
					log.Error().Err(err).Msg("failed to get dmz IPs")
				}
				ch <- ips
			}
		}
	}()

	return ch
}

func (n *networker) YggAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ips, err := n.ndmz.GetIPFor(yggdrasil.YggIface)
				if err != nil {
					log.Error().Err(err).Str("inf", yggdrasil.YggIface).Msg("failed to get public IPs")
				}
				ch <- ips
			}
		}
	}()

	return ch
}

func (n *networker) PublicAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				ips, err := public.IPs()
				if err != nil {
					log.Error().Err(err).Msg("failed to get public IPs")
				}
				ch <- ips
			}
		}
	}()

	return ch
}

func (n *networker) ZOSAddresses(ctx context.Context) <-chan pkg.NetlinkAddresses {
	// we don't use monitorNS because
	// 1- this is the host namespace
	// 2- the ZOS bridge must exist all the time
	updates := make(chan netlink.AddrUpdate)
	if err := netlink.AddrSubscribe(updates, ctx.Done()); err != nil {
		log.Fatal().Err(err).Msg("failed to listen to netlink address updates")
	}

	link, err := netlink.LinkByName(types.DefaultBridge)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not find the '%s' bridge", types.DefaultBridge)
	}

	get := func() pkg.NetlinkAddresses {
		var result pkg.NetlinkAddresses
		values, _ := netlink.AddrList(link, netlink.FAMILY_ALL)
		for _, value := range values {
			result = append(result, *value.IPNet)
		}

		return result
	}

	addresses := get()

	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-updates:
				if update.LinkIndex != link.Attrs().Index {
					continue
				}

				addresses = get()
			case <-time.After(2 * time.Second):
				ch <- addresses
			}
		}
	}()

	return ch

}

func (n *networker) syncWGPorts() error {
	names, err := namespace.List("n-")
	if err != nil {
		return err
	}

	readPort := func(name string) (int, error) {
		netNS, err := namespace.GetByName(name)
		if err != nil {
			return 0, err
		}
		defer netNS.Close()

		ifaceName := strings.Replace(name, "n-", "w-", 1)

		var port int
		err = netNS.Do(func(_ ns.NetNS) error {
			link, err := wireguard.GetByName(ifaceName)
			if err != nil {
				return err
			}
			d, err := link.Device()
			if err != nil {
				return err
			}

			port = d.ListenPort
			return nil
		})
		if err != nil {
			return 0, err
		}

		return port, nil
	}

	for _, name := range names {
		port, err := readPort(name)
		if err != nil {
			log.Error().Err(err).Str("namespace", name).Msgf("failed to read port for network namespace")
			continue
		}
		//skip error cause we don't care if there are some duplicate at this point
		_ = n.portSet.Add(uint(port))
	}

	return nil
}

// createNetNS create a network namespace and set lo interface up
func createNetNS(name string) (ns.NetNS, error) {
	var netNs ns.NetNS
	var err error
	if namespace.Exists(name) {
		netNs, err = namespace.GetByName(name)
	} else {
		netNs, err = namespace.Create(name)
	}

	if err != nil {
		return nil, fmt.Errorf("fail to create network namespace %s: %w", name, err)
	}

	err = netNs.Do(func(_ ns.NetNS) error {
		return ifaceutil.SetLoUp()
	})

	if err != nil {
		_ = namespace.Delete(netNs)
		return nil, fmt.Errorf("failed to bring lo interface up in namespace %s: %w", name, err)
	}

	return netNs, nil
}

// tapName returns the name of the tap device for a network namespace
func tapName(netID pkg.NetID) (string, error) {
	name := fmt.Sprintf("t-%s", netID)
	if len(name) > 15 {
		return "", errors.Errorf("tap name too long %s", name)
	}
	return name, nil
}

func pubTapName(resID string) (string, error) {
	name := fmt.Sprintf("p-%s", resID)
	if len(name) > 15 {
		return "", errors.Errorf("tap name too long %s", name)
	}
	return name, nil
}

var ulaPrefix = net.IPNet{
	IP:   net.ParseIP("fc00::"),
	Mask: net.CIDRMask(7, 128),
}

func isULA(ip net.IP) bool {
	return ulaPrefix.Contains(ip)
}

var yggPrefix = net.IPNet{
	IP:   net.ParseIP("200::"),
	Mask: net.CIDRMask(7, 128),
}

func isYgg(ip net.IP) bool {
	return yggPrefix.Contains(ip)
}
