package network

import (
	"encoding/binary"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"

	"net"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"
	zosip "github.com/threefoldtech/zosv2/modules/network/ip"
	"github.com/threefoldtech/zosv2/modules/network/macvlan"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
)

const (
	// GWNamespace is the default GW name (BAR ;-) )
	GWNamespace = "gw"
)

// CreateGateway is the main router for the ExitPoints in an Exitnode
func CreateGateway(prefixZero *net.IPNet, n int, allocNr int) error {
	var (
		netNS ns.NetNS
		err   error
	)

	netNS, err = namespace.GetByName(GWNamespace)
	if err != nil {
		netNS, err = namespace.Create(GWNamespace)
	}
	if err != nil {
		return errors.Wrap(err, "failed to create gateway namespace")
	}
	defer netNS.Close()

	pubIface, err := getPublicIface()
	if err != nil {
		return errors.Wrap(err, "failed to find a public interface for the gateway")
	}

	//fmt.Sprintf("pub-%d-%d", n, allocNr),
	m, err := macvlan.Create(
		zosip.GWPubName(allocNr, n),
		pubIface,
		netNS)
	if err != nil {
		return err
	}

	// see https://github.com/threefoldtech/zosv2/blob/master/specs/network/Gateway_Container.md#implementation-details
	// n is the exit node number, we use it to configure the gateway ips and routes deterministically
	ips := []*net.IPNet{
		{
			IP:   net.ParseIP(fmt.Sprintf("fe80::%x", n<<12)),
			Mask: net.CIDRMask(64, 128),
		},
		{
			IP:   gwIP(prefixZero.IP, n),
			Mask: net.CIDRMask(64, 128),
		},
	}

	routes := []*netlink.Route{
		{
			Dst: &net.IPNet{
				IP:   net.ParseIP("::"),
				Mask: net.CIDRMask(0, 128),
			},
			Gw:        net.ParseIP("fe80::1"),
			LinkIndex: m.Attrs().Index,
		},
	}

	if err := macvlan.Install(m, ips, routes, netNS); err != nil {
		return errors.Wrap(err, "failed to configure gateway macvlan")
	}
	return nil
}

func addNR2GW(prefixZero, nrPrefix *net.IPNet, exitNodeNr int, allocNr int8) error {
	n, err := zosip.NewNibble(nrPrefix, allocNr)

	gwNS, err := namespace.GetByName(GWNamespace)
	if err != nil {
		return errors.Wrap(err, "gateway namespace doesn't not exist yet")
	}
	defer gwNS.Close()

	nrNamespace := n.NamespaceName()
	nrNS, err := namespace.GetByName(nrNamespace)
	if err != nil {
		return errors.Wrap(err, "gateway namespace doesn't exist yet")
	}
	defer nrNS.Close()

	var gwVethName string
	err = nrNS.Do(func(_ ns.NetNS) error {
		if err := ifaceutil.SetLoUp(); err != nil {
			return err
		}

		log.Info().
			Str("namespace", nrNamespace).
			Str("veth", "eth0").
			Msg("Create veth pair in net namespace")

		// create the veth pair inside the NR namespace and move one side into the gateway namespace
		_, _, err := ip.SetupVeth(n.GWtoEPName(), 1500, gwNS)
		if err != nil {
			return errors.Wrapf(err, "failed to create gateway veth pair in namespace (%s)", nrNamespace)
		}

		// gwVethName = gwVeth.Name

		// link, err := netlink.LinkByName(containerVeth.Name)
		// if err != nil {
		// 	return err
		// }

		// TODO: add ip or route ?

		return nil
	})
	if err != nil {
		return err
	}

	err = gwNS.Do(func(_ ns.NetNS) error {
		nrVeth, err := netlink.LinkByName(gwVethName)
		if err != nil {
			return err
		}

		ips := []*net.IPNet{
			{
				IP:   net.ParseIP(fmt.Sprintf("fe80::%x:%s", exitNodeNr, n.Hex())),
				Mask: net.CIDRMask(64, 128),
			},
			{
				IP:   nrIP(prefixZero.IP, nrPrefix.IP, exitNodeNr),
				Mask: net.CIDRMask(64, 128),
			},
		}

		for _, ip := range ips {
			if err := netlink.AddrAdd(nrVeth, &netlink.Addr{IPNet: ip}); err != nil {
				log.Error().
					Str("addr", ip.String()).
					Str("link", nrVeth.Attrs().Name).
					Err(err).Msg("failed to set address on link")
				return err
			}
		}

		route := &netlink.Route{
			Dst: &net.IPNet{
				IP:   net.ParseIP(fmt.Sprintf("%x:%s::", prefixZero.IP[4:6], n.Hex())),
				Mask: net.CIDRMask(64, 128),
			},
			Gw:        net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex())),
			LinkIndex: nrVeth.Attrs().Index,
		}
		if err = netlink.RouteAdd(route); err != nil {
			log.Error().
				Str("route", route.String()).
				Str("link", nrVeth.Attrs().Name).
				Err(err).Msg("failed to set route on link")
			return err
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to configure macvlan gateway interface")
	}

	// TODO:  is this required ?
	// if _, err := sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.disable_ipv6", hostVeth.Attrs().Name), "1"); err != nil {
	// 	return errors.Wrapf(err, "failed to disable ip6 on bridge %s", hostVeth.Attrs().Name)
	// }

	return nil
}

func gwIP(prefix net.IP, n int) net.IP {
	b := make([]byte, net.IPv6len)
	copy(b, prefix[:6])
	binary.BigEndian.PutUint16(b[6:], uint16(n<<12))
	b[net.IPv6len-1] = 0x001

	return net.IP(b)
}

func nrIP(prefixZero, nrPrefix net.IP, n int) net.IP {
	b := make([]byte, net.IPv6len)
	copy(b, prefixZero[:6])
	binary.BigEndian.PutUint16(b[12:14], uint16(n<<12))
	copy(b[14:], nrPrefix[6:8])

	return net.IP(b)
}
