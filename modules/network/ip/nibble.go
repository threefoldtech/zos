package ip

import (
	"fmt"
	"net"
)

// Nibble is an helper struct used to generate
// deterministic name based on an IP address
type Nibble struct {
	allocNr   int8
	hexNibble string
}

// NewNibble create a new Nibble object
func NewNibble(prefix net.IPNet, allocNr int8) *Nibble {

	var b []byte
	size, _ := prefix.Mask.Size()
	if size <= 40 {
		b = []byte(prefix.IP)[4:8]
	} else {
		b = []byte(prefix.IP)[6:8]
	}

	return &Nibble{
		hexNibble: fmt.Sprintf("%x", b),
		allocNr:   allocNr,
	}
}

// WiregardName return the deterministic wireguard name
func (n *Nibble) WiregardName() string {
	return fmt.Sprintf("wg-%s-%d", n.hexNibble, n.allocNr)
}

// BridgeName return the deterministic bridge name
func (n *Nibble) BridgeName() string {
	return fmt.Sprintf("br-%s-%d", n.hexNibble, n.allocNr)
}

// NetworkName return the deterministic network name
func (n *Nibble) NetworkName() string {
	return fmt.Sprintf("net-%s-%d", n.hexNibble, n.allocNr)
}

// VethName return the deterministic veth interface name
func (n *Nibble) VethName() string {
	return fmt.Sprintf("veth-%s-%d", n.hexNibble, n.allocNr)
}
