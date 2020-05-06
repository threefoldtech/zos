package pkg

import (
	"fmt"
	"net"
)

//go:generate zbusc -module vmd -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+VMModule stubs/vmd_stub.go

// VMNetworkInfo structure
type VMNetworkInfo struct {
	// Tap device name
	Tap string
	// Mac address of the device
	MAC string
	// Address of the device in the form of cidr
	AddressCIDR net.IPNet
	// Gateway gateway address
	GatewayIP net.IP
	// Nameservers dns servers
	Nameservers []net.IP
}

// VMDisk specifies vm disk params
type VMDisk struct {
	// Size is disk size in Mib
	Path     string
	ReadOnly bool
	Root     bool
}

// VM config interface
type VM interface {
	// first start by including all public functions that are needed
	// for the firecracker vm to do it's things
	// after that, we can look how we need to change these functions
	// so they work with qemu and firecracker (hypervisor independent)
	GetDisks() []VMDisk
	GetNetwork() VMNetworkInfo
	GetName() string
	Validate() error
	GetKernelArgs() string
	GetKernelImage() string
	GetInitrdImage() string
	GetCPU() uint8
	GetMemory() int64
}

type FirecrackerVM struct {
	// virtual machine name, or ID
	Name string
	// CPU is number of cores assigned to the VM
	CPU uint8
	// Memory size in Mib
	Memory int64
	// Network is network info
	Network VMNetworkInfo
	// KernelImage path to uncompressed linux kernel ELF
	KernelImage string
	// InitrdImage (optiona) path to initrd disk
	InitrdImage string
	// KernelArgs to override the default kernel arguments. (default: "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules")
	KernelArgs string
	// Disks are a list of disks that are going to
	// be auto allocated on the provided storage path
	Disks []VMDisk
}

func (fcvm *FirecrackerVM) GetName() string {
	return fcvm.Name
}

func (fcvm *FirecrackerVM) GetDisks() []VMDisk {
	return fcvm.Disks
}

func (fcvm *FirecrackerVM) GetKernelArgs() string {
	return fcvm.KernelArgs
}

func (fcvm *FirecrackerVM) GetNetwork() VMNetworkInfo {
	// function to parse the Firecracker Network to generic network model (not implemented yet)
	return fcvm.Network
}

func (fcvm *FirecrackerVM) GetKernelImage() string {
	return fcvm.KernelImage
}

func (fcvm *FirecrackerVM) GetInitrdImage() string {
	return fcvm.InitrdImage
}

func (fcvm *FirecrackerVM) GetCPU() uint8 {
	return fcvm.CPU
}

func (fcvm *FirecrackerVM) GetMemory() int64 {
	return fcvm.Memory
}

// Validate Firecracker vm data
func (fcvm *FirecrackerVM) Validate() error {
	missing := func(s string) bool {
		return len(s) == 0
	}

	if missing(fcvm.Name) {
		return fmt.Errorf("name is required")
	}

	if missing(fcvm.KernelImage) {
		return fmt.Errorf("kernel-image is required")
	}

	if fcvm.Memory < 512 {
		return fmt.Errorf("invalid memory must not be less than 512M")
	}

	if fcvm.CPU == 0 || fcvm.CPU > 32 {
		return fmt.Errorf("invalid cpu must be between 1 and 32")
	}

	return nil
}

type QemuVM struct {
	// virtual machine name, or ID
	Name string
	// CPU is number of cores assigned to the VM
	CPU uint8
	// Memory size in Mib
	Memory int64
	// Disks are a list of disks that are going to
	// be auto allocated on the provided storage path
	Disks []VMDisk
}

func (qvm *QemuVM) GetName() string {
	return qvm.Name
}

func (qvm *QemuVM) GetCPU() uint8 {
	return qvm.CPU
}

func (qvm *QemuVM) GetDisks() []VMDisk {
	return qvm.Disks
}

func (qvm *QemuVM) GetMemory() int64 {
	return qvm.Memory
}

func (qvm *QemuVM) GetNetwork() VMNetworkInfo {
	return VMNetworkInfo{}
}

func (qvm *QemuVM) GetKernelArgs() string {
	return ""
}

func (qvm *QemuVM) GetKernelImage() string {
	return ""
}

func (qvm *QemuVM) GetInitrdImage() string {
	return ""
}

// Validate Qemu vm data
func (qvm *QemuVM) Validate() error {
	missing := func(s string) bool {
		return len(s) == 0
	}

	if missing(qvm.Name) {
		return fmt.Errorf("name is required")
	}

	if qvm.Memory < 512 {
		return fmt.Errorf("invalid memory must not be less than 512M")
	}

	if qvm.CPU == 0 || qvm.CPU > 32 {
		return fmt.Errorf("invalid cpu must be between 1 and 32")
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
	Exists(id string) bool
}
