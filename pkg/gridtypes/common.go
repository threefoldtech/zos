package gridtypes

import (
	"fmt"
	"net"
)

// ID is a generic ID type
type ID string

func (id ID) String() string {
	return string(id)
}

// IsEmpty checks if id is empty
func (id ID) IsEmpty() bool {
	return len(id) == 0
}

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType string

// Known device types
const (
	SSDDevice DeviceType = "ssd"
	HDDDevice DeviceType = "hdd"
)

// IPNet type
type IPNet struct{ net.IPNet }

// NewIPNet creates a new IPNet from net.IPNet
func NewIPNet(n *net.IPNet) IPNet {
	return IPNet{IPNet: *n}
}

// ParseIPNet parse iprange
func ParseIPNet(txt string) (r IPNet, err error) {
	if len(txt) == 0 {
		//empty ip net value
		return r, nil
	}
	//fmt.Println("parsing: ", string(text))
	ip, net, err := net.ParseCIDR(txt)
	if err != nil {
		return r, err
	}

	net.IP = ip
	r.IPNet = *net
	return
}

// MustParseIPNet prases iprange, panics if invalid
func MustParseIPNet(txt string) IPNet {
	r, err := ParseIPNet(txt)
	if err != nil {
		panic(err)
	}
	return r
}

// UnmarshalText loads IPRange from string
func (i *IPNet) UnmarshalText(text []byte) error {
	v, err := ParseIPNet(string(text))
	if err != nil {
		return err
	}

	i.IPNet = v.IPNet
	return nil
}

// MarshalJSON dumps iprange as a string
func (i IPNet) MarshalJSON() ([]byte, error) {
	if len(i.IPNet.IP) == 0 {
		return []byte(`""`), nil
	}
	v := fmt.Sprint("\"", i.String(), "\"")
	return []byte(v), nil
}

func (i IPNet) String() string {
	return i.IPNet.String()
}

// Nil returns true if IPNet is not set
func (i *IPNet) Nil() bool {
	return i.IP == nil && i.Mask == nil
}
