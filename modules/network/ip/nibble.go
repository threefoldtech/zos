package ip

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// Nibble is an helper struct used to generate
// deterministic name based on an IPv6 address
type Nibble struct {
	allocNr int8
	nibble  []byte
}

// NewNibble create a new Nibble object
func NewNibble(prefix *net.IPNet, allocNr int8) (*Nibble, error) {

	if prefix == nil || prefix.IP == nil || prefix.Mask == nil {
		return nil, fmt.Errorf("prefix cannot be nil")
	}
	size, _ := prefix.Mask.Size()
	if size != 48 {
		return nil, fmt.Errorf("allocation prefix can only be a /48")
	}
	if allocNr < 0 {
		return nil, fmt.Errorf("allocNr cannot be negative")
	}

	return &Nibble{
		nibble:  []byte(prefix.IP)[6:8],
		allocNr: allocNr,
	}, nil
}

// Hex return the hexadecimal version of the meaningfull nibble
func (n *Nibble) Hex() string {
	return fmt.Sprintf("%x", n.nibble)
}

// WGName return the deterministic wireguard name in the Network Resource
func (n *Nibble) WGName() string {
	return fmt.Sprintf("wg-%s-%d", n.Hex(), n.allocNr)
}

// WireguardPort return the deterministic wireguard listen port
func (n *Nibble) WireguardPort() uint16 {
	return binary.BigEndian.Uint16(n.nibble)
}

// BridgeName return the deterministic bridge name of the Network Resource
func (n *Nibble) BridgeName() string {
	return fmt.Sprintf("br-%s-%d", n.Hex(), n.allocNr)
}

// NamespaceName return the deterministic Namespace name
func (n *Nibble) NamespaceName() string {
	return fmt.Sprintf("net-%s-%d", n.Hex(), n.allocNr)
}

// VethName return the deterministic veth interface name
func (n *Nibble) VethName() string {
	return fmt.Sprintf("veth-%s-%d", n.Hex(), n.allocNr)
}

// NRLocalName return the deterministic veth interface name
// added here for compliance to docs
func (n *Nibble) NRLocalName() string {
	return fmt.Sprintf("veth-%s-%d", n.Hex(), n.allocNr)
}

// EPPubName return the deterministic public interface name
// this Interface points to the veth peer GWtoEPName
func (n *Nibble) EPPubName() string {
	return fmt.Sprintf("pub-%s-%d", n.Hex(), n.allocNr)
}

// EPPubLL ExitPoint Public Link-Local
// the interface that faces the other side of the veth into the GW
// we differentiate it by shifting 2 bytes, having 0001 in the last 2
func (n *Nibble) EPPubLL() *net.IPNet {
	return &net.IPNet{
		IP:   net.ParseIP(fmt.Sprintf("fe80::%s:1", n.Hex())),
		Mask: net.CIDRMask(64, 128),
	}
}

/* // ExitPrefixZero (not needed any more )
func (n *Nibble) ExitPrefixZero(prefix *net.IPNet) *net.IPNet {
	ip := prefix.IP
	ip[14] = n.nibble[0]
	ip[15] = n.nibble[1]
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}
} */

// NRLocalIP4 returns the IPv4 address of a network resource
func (n *Nibble) NRLocalIP4() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, n.nibble[0], n.nibble[1], 1),
		Mask: net.CIDRMask(24, 32),
	}
}

// WGAllowedIP returns the IPv4 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGAllowedIP() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, 255, n.nibble[0], n.nibble[1]),
		Mask: net.CIDRMask(16, 32),
	}
}

// WGAllowedFE80 returns the fe80 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGAllowedFE80() *net.IPNet {
	_, ipnet, _ := net.ParseCIDR(fmt.Sprintf("fe80::%s/128", n.Hex()))
	return ipnet
}

// WGLL returns the fe80 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGLL() net.IP {
	return net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex()))
}

//RouteIPv6Exit (to be renamed) is the gateway in an NR for ::
//that is: the route to the ExitPoint
func (n *Nibble) RouteIPv6Exit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		Gw: net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex())),
	}
}

// RouteIPv4Exit (to be renamed) adds the route for another NR
func (n *Nibble) RouteIPv4Exit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("10.%d.%d.0", n.nibble[0], n.nibble[1])),
			Mask: net.CIDRMask(24, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", n.nibble[0], n.nibble[1])),
	}
}

// RouteIPv4DefaultExit (to be renamed) is the gateway in an NR for ::
// that is: the route to the ExitPoint
func (n *Nibble) RouteIPv4DefaultExit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", n.nibble[0], n.nibble[1])),
	}
}

// EPToGWName return the deterministic nic name of the EXitPoint NR to the gateway
func (n *Nibble) EPToGWName() string {
	return fmt.Sprintf("to-%s-%d", n.Hex(), n.allocNr)
}

// ToGWLinkLocal is the LL ip of the veth pair in the ExitPoint that points
// to the GW
/* func (n *Nibble) GWLinkLocal(exitnodenr int8) net.IP {
	b := make([]byte, net.IPv6len)
	b[0] = 0xfe
	b[1] = 0x80
	binary.BigEndian.PutUint16(b[6:8], uint16(uint16(nr)<<12))
	return net.IP(b)
} */

// GWPubName return the deterministic public iface name for the GW
func GWPubName(exitnodenr, allocnr int) string {
	return fmt.Sprintf("pub-%d-%d", exitnodenr, allocnr)
}

// GWPubIP6 is the IP on the prefixzero segment of the GW container (BAR)
// it returns the list of IP that need to be installed on
// GWPubName() interface.
// that is :
//  - prefix:ExitNodeNr()::1
func GWPubIP6(prefix net.IP, exitnodenr int) *net.IPNet {
	b := make([]byte, net.IPv6len)
	copy(b, prefix[:6])
	binary.BigEndian.PutUint16(b[6:], uint16(exitnodenr))
	b[net.IPv6len-1] = 0x001
	return &net.IPNet{IP: net.IP(b), Mask: net.CIDRMask(64, 128)}
}

/* func (n *Nibble) NRToGWLinkLocal() net.IP {
	b := make([]byte, net.IPv6len)
	b[0] = 0xfe
	b[1] = 0x80
	b[4] = n.nibble[0]
	b[5] = n.nibble[1]
	return net.IP(b)
} */

/* func (n *Nibble) GWNRIP(prefixZero net.IP, nr int) net.IP {
	b := make([]byte, net.IPv6len)
	copy(b, prefixZero[:6])
	binary.BigEndian.PutUint16(b[12:14], uint16(nr<<12))
	copy(b[14:], n.nibble)

	return net.IP(b)
} */

// GWPubName is the name of the iface facing the penultimate router
// format pub-X-Y
// where X =  exitnode nr, Y = allocationnr
func (n *Nibble) GWPubName(exitnodenr int) string {
	return fmt.Sprintf("pub-%x-%d", exitnodenr, n.allocNr)
}

// GWPubLL is an added link-local address of the iface facint the router
// Format fe80::X:0:0:0:1/64 and
//     $prefix:X:0:0:0:1/64
// where X = exitnodenr
func GWPubLL(exitnodenr int) *net.IPNet {
	return &net.IPNet{
		IP:   net.ParseIP(fmt.Sprintf("fe80::%x:0:0:0:1", exitnodenr)),
		Mask: net.CIDRMask(64, 128),
	}

}

// GWtoEPName is the gw Container interface facing the Exitpoint veth peer
func (n *Nibble) GWtoEPName() string {
	return fmt.Sprintf("to-%s-%d", n.Hex(), n.allocNr)
}

// GWtoEPLL is the link-local address on the iface in the wg facing the
// Exitpoint's pub iface (veth pair)
func (n *Nibble) GWtoEPLL() *net.IPNet {
	return &net.IPNet{
		IP:   net.ParseIP(fmt.Sprintf("fe80::1:%s", n.Hex())),
		Mask: net.CIDRMask(64, 128),
	}
}
