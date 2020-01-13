package pkg

import (
	"net"
)

// VMNetworkInfo structure
type VMNetworkInfo struct {
	NetworkInfo
	MAC net.HardwareAddr
}

// VM config structure
type VM struct {
	// virtual machine name, or ID
	Name string
	// CPU is number of cores assigned to the VM
	CPU uint8
	// Memory size
	Memory uint64
	// Network is network info
	Network VMNetworkInfo
	// an allocated storage for the vm operation
	// where files/virtual disks can be allocated
	Storage string
}

// VMModule defines the virtual machine module interface
type VMModule interface {
	Run(vm VM)
	Inspect(name string)
	Delete(name string)
}
