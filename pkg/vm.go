package pkg

import (
	"fmt"
	"net"
	"os"
)

//go:generate zbusc -module vmd -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+VMModule stubs/vmd_stub.go

// VMNetworkInfo structure
type VMNetworkInfo struct {
	NetworkInfo
	MAC net.HardwareAddr
}

// Disk specifies vm disk params
type Disk struct {
	// Size is disk size in Mib
	Size uint64
}

// VM config structure
type VM struct {
	// virtual machine name, or ID
	Name string
	// CPU is number of cores assigned to the VM
	CPU uint8
	// Memory size in Mib
	Memory int64
	// Network is network info
	Network VMNetworkInfo
	// an allocated storage for the vm operation
	// where files/virtual disks can be allocated
	Storage string
	// KernelImage path to uncompressed linux kernel ELF
	KernelImage string
	// KernelArgs to override the default kernel arguments. (default: "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules")
	KernelArgs string
	// RootImage path to root disk image
	RootImage string
	// Disks are a list of disks that are going to
	// be auto allocated on the provided storage path
	Disks []Disk
}

// Validate vm data
func (vm *VM) Validate() error {
	missing := func(s string) bool {
		return len(s) == 0
	}

	if missing(vm.Name) {
		return fmt.Errorf("name is required")
	}

	if missing(vm.KernelImage) {
		return fmt.Errorf("kernel-image is required")
	}

	if missing(vm.RootImage) {
		return fmt.Errorf("root-image is required")
	}

	if missing(vm.Storage) {
		return fmt.Errorf("storage is required")
	}

	if vm.Memory <= 0 {
		return fmt.Errorf("invalid memory must be bigger than 0")
	}

	if vm.CPU == 0 || vm.CPU > 32 {
		return fmt.Errorf("invalid cpu must be between 1 and 32")
	}

	if stat, err := os.Stat(vm.Storage); err != nil || !stat.IsDir() {
		return fmt.Errorf("storage must exist and must be a directory")
	}

	return nil
}

// VMInfo returned by the inspect method
type VMInfo struct {
	// Flag for enabling/disabling Hyperthreading
	// Required: true
	HtEnabled bool

	// Memory size of VM
	// Required: true
	Memory int64

	// Number of vCPUs (either 1 or an even number)
	CPU int64
}

// VMModule defines the virtual machine module interface
type VMModule interface {
	Run(vm VM) error
	Inspect(name string) (VMInfo, error)
	Delete(name string) error
}
