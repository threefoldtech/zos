package pkg

import (
	"context"
	"net"
)

type Route struct {
	Net net.IPNet
	// Gateway can be nil, in that
	// case the device is used as a dev instead
	Gateway net.IP
}

//go:generate mkdir -p stubs
//go:generate zbusc -module netlight -version 0.0.1 -name netlight -package stubs github.com/threefoldtech/zos4/pkg+NetworkerLight stubs/network_light_stub.go

// NetworkerLight is the interface for the network light module
type NetworkerLight interface {
	Create(name string, privateNet net.IPNet, seed []byte) error
	Delete(name string) error
	AttachPrivate(name, id string, vmIp net.IP) (device TapDevice, err error)
	AttachMycelium(name, id string, seed []byte) (device TapDevice, err error)
	Detach(id string) error
	Interfaces(iface string, netns string) (Interfaces, error)
	AttachZDB(id string) (string, error)
	ZDBIPs(namespace string) ([]net.IP, error)
	Namespace(id string) string
	Ready() error
	ZOSAddresses(ctx context.Context) <-chan NetlinkAddresses
	GetSubnet(networkID NetID) (net.IPNet, error)
}

type TapDevice struct {
	Name   string
	Mac    net.HardwareAddr
	IP     *net.IPNet
	Routes []Route
}

// Interfaces struct to bypass zbus generation error
// where it generate a stub with map as interface instead of map
type Interfaces struct {
	Interfaces map[string]Interface
}

type Interface struct {
	Name string
	IPs  []net.IPNet
	Mac  string
}
