package primitives

import (
	"context"
	"net"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
)

// QemuResult result returned by qemu reservation
type QemuResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// Qemu reservation data
type Qemu struct {
	// NetworkID of the network namepsace in which to run the VM. The network
	// must be provisioned previously.
	NetworkID pkg.NetID `json:"network_id"`
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IP `json:"ip"`
	// Image of the VM.
	Image string `json:"image"`
	// QemuCapacity is the amount of resource to allocate to the virtual machine
	Capacity QemuCapacity `json:"capacity"`
}

// QemuCapacity is the amount of resource to allocate to the virtual machine
type QemuCapacity struct {
	// Number of CPU
	CPU uint `json:"cpu"`
	// Memory in MiB
	Memory uint64 `json:"memory"`
	// HDD in GB
	HDDSize uint64 `json:"hdd"`
}

const qemuFlistURL = "https://hub.grid.tf/maximevanhees.3bot/qemu-flist-tarball.flist"

func (p *Provisioner) qemuProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.qemuProvisionImpl(ctx, reservation)
}

func (p *Provisioner) qemuProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result QemuResult, err error) {
	return result, err
}

func (p *Provisioner) qemuDecommision(ctx context.Context, reservation *provision.Reservation) error {
	return nil
}
