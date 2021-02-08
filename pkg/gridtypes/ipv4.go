package gridtypes

import (
	"fmt"
)

// PublicIP structure
type PublicIP struct {
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP IPNet `json:"ip"`
}

// Valid validate public ip input
func (p *PublicIP) Valid() error {
	if len(p.IP.IP) == 0 {
		return fmt.Errorf("empty ip value")
	}

	if p.IP.IP.To4() == nil {
		return fmt.Errorf("invalid ip format")
	}

	return nil
}

// PublicIPResult result returned by publicIP reservation
type PublicIPResult struct {
	IP IPNet `json:"ip"`
}
