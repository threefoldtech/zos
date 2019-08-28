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

// WiregardName return the deterministic wireguard name
func (n *Nibble) WiregardName() string {
	return fmt.Sprintf("wg-%s-%d", n.Hex(), n.allocNr)
}

// WireguardPort return the deterministic wireguard listen port
func (n *Nibble) WireguardPort() uint16 {
	return binary.BigEndian.Uint16(n.nibble)
}

// BridgeName return the deterministic bridge name
func (n *Nibble) BridgeName() string {
	return fmt.Sprintf("br-%s-%d", n.Hex(), n.allocNr)
}

// NetworkName return the deterministic network name
func (n *Nibble) NetworkName() string {
	return fmt.Sprintf("net-%s-%d", n.Hex(), n.allocNr)
}

// VethName return the deterministic veth interface name
func (n *Nibble) VethName() string {
	return fmt.Sprintf("veth-%s-%d", n.Hex(), n.allocNr)
}

// PubName return the deterministic public interface name
func (n *Nibble) PubName() string {
	return fmt.Sprintf("pub-%s-%d", n.Hex(), n.allocNr)
}

func (n *Nibble) ExitFe80() *net.IPNet {
	return &net.IPNet{
		IP:   net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex())),
		Mask: net.CIDRMask(64, 128),
	}
}

func (n *Nibble) ExitPrefixZero(prefix *net.IPNet) *net.IPNet {
	ip := prefix.IP
	ip[14] = n.nibble[0]
	ip[15] = n.nibble[1]
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(64, 128),
	}
}

// NRIPv4 returns the IPv4 address of a network resource
func (n *Nibble) NRIPv4() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, n.nibble[0], n.nibble[1], 1),
		Mask: net.CIDRMask(24, 32),
	}
}

// WGAllowedIP returns the IPv4 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGIP() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, 255, n.nibble[0], n.nibble[1]),
		Mask: net.CIDRMask(16, 32),
	}
}

// WGAllowedIP returns the IPv4 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGAllowedIP() *net.IPNet {
	return &net.IPNet{
		IP:   net.IPv4(10, 255, n.nibble[0], n.nibble[1]),
		Mask: net.CIDRMask(24, 32),
	}
}

// WGAllowedFE80 returns the fe80 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGAllowedFE80() *net.IPNet {
	_, ipnet, _ := net.ParseCIDR(fmt.Sprintf("fe80::%s/128", n.Hex()))
	return ipnet
}

// WGRouteGateway returns the fe80 address to be used in wireguard allowed ip configuration
func (n *Nibble) WGRouteGateway() net.IP {
	return net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex()))
}

func (n *Nibble) RouteIPv6Exit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("::"),
			Mask: net.CIDRMask(0, 128),
		},
		Gw: net.ParseIP(fmt.Sprintf("fe80::%s", n.Hex())),
	}
}

func (n *Nibble) RouteIPv4Exit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("10.%d.%d.0", n.nibble[0], n.nibble[1])),
			Mask: net.CIDRMask(24, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", n.nibble[0], n.nibble[1])),
	}
}

func (n *Nibble) RouteIPv4DefaultExit() *netlink.Route {
	return &netlink.Route{
		Dst: &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(0, 32),
		},
		Gw: net.ParseIP(fmt.Sprintf("10.255.%d.%d", n.nibble[0], n.nibble[1])),
	}
}

// ToGWName return the deterministic nic name of the NR to the gateway
func (n *Nibble) ToGWName() string {
	return fmt.Sprintf("to-%s-%d", n.Hex(), n.allocNr)
}

// GWtoNRName return the deterministic name for a nic in the gw to the NR
func (n *Nibble) GWtoNRName() string {
	return fmt.Sprintf("nr-%s-%d", n.Hex(), n.allocNr)
}

func (n *Nibble) GWLinkLocal(exitnodenr int8) net.IP {
	b := make([]byte, net.IPv6len)
	b[0] = 0xfe
	b[1] = 0x80
	binary.BigEndian.PutUint16(b[6:8], uint16(nr<<12))
	return net.IP(b)
}

func (n *Nibble) GWIP(prefix net.IP, nr int8) net.IP {
	b := make([]byte, net.IPv6len)
	copy(b, prefix[:6])
	binary.BigEndian.PutUint16(b[6:], uint16(nr<<12))
	b[net.IPv6len-1] = 0x001
	return net.IP(b)
}

func (n *Nibble) GWNRLinkLocal() net.IP {
	b := make([]byte, net.IPv6len)
	b[0] = 0xfe
	b[1] = 0x80
	b[4] = n.nibble[0]
	b[5] = n.nibble[1]
	return net.IP(b)
}

func (n *Nibble) GWNRIP(prefixZero net.IP, nr int) net.IP {
	b := make([]byte, net.IPv6len)
	copy(b, prefixZero[:6])
	binary.BigEndian.PutUint16(b[12:14], uint16(nr<<12))
	copy(b[14:], n.nibble)

	return net.IP(b)
}

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
