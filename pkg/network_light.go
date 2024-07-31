package pkg

import (
	"net"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module netlight -version 0.0.1 -name netlight -package stubs github.com/threefoldtech/zos/pkg+NetworkerLight stubs/network_light_stub.go

// NetworkerLight is the interface for the network light module
type NetworkerLight interface {
	Create(name string, privateNet net.IPNet, seed []byte) error
	Delete(name string) error
	AttachPrivate(name, id string, vmIp net.IP) (device TapDevice, err error)
	AttachMycelium(name, id string, seed []byte) (device TapDevice, err error)
	Detach(id string) error
	Interfaces(iface string, netns string) (Interfaces, error)
}

type TapDevice struct {
	Name    string
	Mac     net.HardwareAddr
	IP      *net.IPNet
	Gateway *net.IPNet
}
