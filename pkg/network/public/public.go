package public

import (
	"fmt"
	"net"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/network/ifaceutil"
	"github.com/threefoldtech/zos/pkg/network/macvlan"
	"github.com/threefoldtech/zos/pkg/network/namespace"
	"github.com/threefoldtech/zos/pkg/network/options"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

const (
	publicNsMACDerivationSuffix = "-public"
)

// EnsurePublicBridge makes sure that the public bridge exists
func ensurePublicBridge() (*netlink.Bridge, error) {
	br, err := bridge.Get(types.PublicBridge)
	if err != nil {
		return bridge.New(types.PublicBridge)
	}

	return br, nil
}

// EnsurePublicSetup create the public setup, it's okay to have inf == nil
func EnsurePublicSetup(nodeID pkg.Identifier, inf *types.PubIface) (*netlink.Bridge, error) {
	br, err := ensurePublicBridge()
	if err != nil {
		return nil, err
	}

	// find if we have anything attached to this bridge (so initialization)
	// has been done before.
	all, err := netlink.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list host links")
	}

	filtered := all[:0]
	for _, link := range all {
		if link.Attrs().MasterIndex == br.Index {
			filtered = append(filtered, link)
		}
	}

	if len(filtered) > 0 {
		// the bridge already has connected devices, hence we are
		// sure to a great extend that things has been done before.
		// but to be sure, we can take one extra verification step
		// by checking if one of the links is actually inf.master
		// in case inf is set.

		return br, nil
	}
	var exit string
	if inf == nil {
		// find possible exit interface.
		// or fall back to zos.
		exit, err = findPossibleExit()
		if err != nil {
			return nil, errors.Wrap(err, "failed to find possible exit")
		}
	} else {
		exit = inf.Master
	}

	exitLink, err := netlink.LinkByName(exit)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get link '%s' by name", exit)
	}

	if err := netlink.LinkSetUp(exitLink); err != nil {
		return nil, errors.Wrapf(err, "failed to set link '%s' up", exitLink.Attrs().Name)
	}

	if err := bridge.Attach(exitLink, br); err != nil {
		return nil, errors.Wrap(err, "failed to attach exit nic to public bridge 'br-pub'")
	}

	if inf != nil {
		if err := createPublicNS(nodeID, inf); err != nil {
			return nil, errors.Wrap(err, "failed to ensure public namespace setup")
		}
	}

	return br, netlink.LinkSetUp(br)

}

func findPossibleExit() (string, error) {
	links, err := bootstrap.AnalyseLinks(
		bootstrap.PhysicalFilter,
		bootstrap.PluggedFilter,
		bootstrap.NotAttachedFilter,
	)

	if err != nil {
		return "", errors.Wrap(err, "failed to analyse links")
	}

	for _, link := range links {
		for _, addr := range link.Addrs6 {
			if addr.IP.IsGlobalUnicast() && !ifaceutil.IsULA(addr.IP) {
				return link.Name, nil
			}
		}
	}

	return types.DefaultBridge, nil
}

func ensureNamespace() (ns.NetNS, error) {
	if !namespace.Exists(types.PublicNamespace) {
		log.Info().Str("namespace", types.PublicNamespace).Msg("Create network namespace")
		return namespace.Create(types.PublicNamespace)
	}

	return namespace.GetByName(types.PublicNamespace)
}

func ensurePublicMacvlan(iface *types.PubIface, pubNS ns.NetNS) (*netlink.Macvlan, error) {
	var (
		pubIface *netlink.Macvlan
		err      error
	)

	if !ifaceutil.Exists(types.PublicIface, pubNS) {

		switch iface.Type {
		case types.MacVlanIface:
			pubIface, err = macvlan.Create(types.PublicIface, types.PublicBridge, pubNS)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create public mac vlan interface")
			}

		default:
			return nil, fmt.Errorf("unsupported public interface type %s", iface.Type)
		}

	} else {
		err := pubNS.Do(func(_ ns.NetNS) error {
			pubIface, err = macvlan.GetByName(types.PublicIface)
			return err
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get public macvlan interface")
		}
	}

	return pubIface, nil
}

func publicConfig(iface *types.PubIface) (ips []*net.IPNet, routes []*netlink.Route, err error) {
	if !iface.IPv6.Nil() && iface.GW6 != nil {
		routes = append(routes, &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw: iface.GW6,
		})
		ips = append(ips, &iface.IPv6.IPNet)
	}

	if !iface.IPv4.Nil() && iface.GW4 != nil {
		routes = append(routes, &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP("0.0.0.0"),
				Mask: net.CIDRMask(0, 32),
			},
			Gw: iface.GW4,
		})
		ips = append(ips, &iface.IPv4.IPNet)
	}

	if len(ips) <= 0 || len(routes) <= 0 {
		err := fmt.Errorf("missing some information in the exit iface object")
		log.Error().Err(err).Msg("failed to configure public interface")
		return nil, nil, err
	}

	return ips, routes, nil
}

// createPublicNS creates a public namespace in a node
func createPublicNS(nodeID pkg.Identifier, iface *types.PubIface) error {

	pubNS, err := ensureNamespace()
	if err != nil {
		return err
	}

	defer pubNS.Close()

	pubIface, err := ensurePublicMacvlan(iface, pubNS)
	if err != nil {
		return err
	}

	log.Info().
		Str("pub iface", fmt.Sprintf("%+v", pubIface)).
		Msg("configure public interface inside public namespace")

	ips, routes, err := publicConfig(iface)
	if err != nil {
		return errors.Wrap(err, "failed configure public network interface")
	}

	mac := ifaceutil.HardwareAddrFromInputBytes([]byte(nodeID.Identity() + publicNsMACDerivationSuffix))
	if err := macvlan.Install(pubIface, mac, ips, routes, pubNS); err != nil {
		return err
	}

	err = pubNS.Do(func(_ ns.NetNS) error {
		if err := options.IPv6AcceptRA(options.RAAcceptIfForwardingIsEnabled); err != nil {
			return errors.Wrapf(err, "failed to accept_ra=2 in public namespace")
		}

		if err := options.IPv6LearnDefaultRouteInRA(true); err != nil {
			return errors.Wrapf(err, "failed to enable enable_defrtr=1 in public namespace")
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "error while configuring IPv6 public namespace")
	}

	return nil
}
