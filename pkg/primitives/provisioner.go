package primitives

import (
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Primitives hold all the logic responsible to provision and decomission
// the different primitives workloads defined by this package
type Primitives struct {
	provision.Provisioner
	zbus zbus.Client
}

var _ provision.Provisioner = (*Primitives)(nil)

// NewPrimitivesProvisioner creates a new 0-OS provisioner
func NewPrimitivesProvisioner(zbus zbus.Client) *Primitives {
	p := &Primitives{
		zbus: zbus,
	}

	provisioners := map[gridtypes.WorkloadType]provision.DeployFunction{
		zos.ZMountType:           p.zMountProvision,
		zos.NetworkType:          p.networkProvision,
		zos.ZDBType:              p.zdbProvision,
		zos.ZMachineType:         p.virtualMachineProvision,
		zos.PublicIPType:         p.publicIPProvision,
		zos.GatewayNameProxyType: p.gwProvision,
		zos.GatewayFQDNProxyType: p.gwFQDNProvision,
	}
	decommissioners := map[gridtypes.WorkloadType]provision.RemoveFunction{
		zos.ZMountType:           p.zMountDecommission,
		zos.NetworkType:          p.networkDecommission,
		zos.ZDBType:              p.zdbDecommission,
		zos.ZMachineType:         p.vmDecomission,
		zos.PublicIPType:         p.publicIPDecomission,
		zos.GatewayNameProxyType: p.gwDecommission,
		zos.GatewayFQDNProxyType: p.gwFQDNDecommission,
	}

	// only network support update atm
	updaters := map[gridtypes.WorkloadType]provision.DeployFunction{
		zos.NetworkType: p.networkProvision,
	}

	p.Provisioner = provision.NewMapProvisioner(provisioners, decommissioners, updaters)

	return p
}
