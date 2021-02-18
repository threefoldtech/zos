package types

import (
	"fmt"
	"net"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// IfaceType define the different public interface supported
type IfaceType string

const (
	//VlanIface means we use vlan for the public interface
	VlanIface IfaceType = "vlan"
	//MacVlanIface means we use macvlan for the public interface
	MacVlanIface IfaceType = "macvlan"
)

// MacAddress type
type MacAddress struct{ net.HardwareAddr }

// MarshalText marshals MacAddress type to a string
func (mac MacAddress) MarshalText() ([]byte, error) {
	if mac.HardwareAddr == nil {
		return nil, nil
	} else if mac.HardwareAddr.String() == "" {
		return nil, nil
	}
	return []byte(mac.HardwareAddr.String()), nil
}

// UnmarshalText loads a macaddress from a string
func (mac *MacAddress) UnmarshalText(addr []byte) error {
	if len(addr) == 0 {
		return nil
	}
	addr, err := net.ParseMAC(string(addr))
	if err != nil {
		return err
	}
	mac.HardwareAddr = addr
	return nil
}

// IfaceInfo is the information about network interfaces
// that the node will publish publicly
// this is used to be able to configure public side of a node
type IfaceInfo struct {
	Name       string            `json:"name"`
	Addrs      []gridtypes.IPNet `json:"addrs"`
	Gateway    []net.IP          `json:"gateway"`
	MacAddress MacAddress        `json:"macaddress"`
}

// DefaultIP return the IP address of the interface that has a default gateway configured
// this function currently only check IPv6 addresses
func (i *IfaceInfo) DefaultIP() (net.IP, error) {
	if len(i.Gateway) <= 0 {
		return nil, fmt.Errorf("interface has not gateway")
	}

	for _, addr := range i.Addrs {
		if addr.IP.IsLinkLocalUnicast() ||
			addr.IP.IsLinkLocalMulticast() ||
			addr.IP.To4() != nil {
			continue
		}

		if addr.IP.To16() != nil {
			return addr.IP, nil
		}
	}
	return nil, fmt.Errorf("no ipv6 address with default gateway")
}

// PubIface is the configuration of the interface
// that is connected to the public internet
type PubIface struct {
	// Type define if we need to use
	// the Vlan field or the MacVlan
	Type IfaceType `json:"type"`
	Vlan int16     `json:"vlan"`
	// Macvlan net.HardwareAddr

	IPv4 gridtypes.IPNet `json:"ipv4"`
	IPv6 gridtypes.IPNet `json:"ipv6"`

	GW4 net.IP `json:"gw4"`
	GW6 net.IP `json:"gw6"`

	Version int `json:"version"`
}
