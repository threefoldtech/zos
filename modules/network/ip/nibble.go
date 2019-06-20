package ip

import (
	"fmt"
	"net"
)

type Nibble struct {
	allocNr   int8
	hexNibble string
}

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

func (n *Nibble) WiregardName() string {
	return fmt.Sprintf("wg-%s-%d", n.hexNibble, n.allocNr)
}
func (n *Nibble) BridgeName() string {
	return fmt.Sprintf("br-%s-%d", n.hexNibble, n.allocNr)
}
func (n *Nibble) NetworkName() string {
	return fmt.Sprintf("net-%s-%d", n.hexNibble, n.allocNr)
}
func (n *Nibble) VethName() string {
	return fmt.Sprintf("veth-%s-%d", n.hexNibble, n.allocNr)
}
