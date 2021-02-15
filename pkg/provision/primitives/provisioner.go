package primitives

import (
	"context"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type provisionFn func(ctx context.Context, wl *gridtypes.Workload) (interface{}, error)
type decommissionFn func(ctx context.Context, wl *gridtypes.Workload) error

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
		gridtypes.ContainerType:  p.containerProvision,
		gridtypes.VolumeType:     p.volumeProvision,
		gridtypes.NetworkType:    p.networkProvision,
		gridtypes.ZDBType:        p.zdbProvision,
		gridtypes.KubernetesType: p.kubernetesProvision,
		gridtypes.PublicIPType:   p.publicIPProvision,
	}
	decommissioners := map[gridtypes.WorkloadType]provision.RemoveFunction{
		gridtypes.ContainerType:  p.containerDecommission,
		gridtypes.VolumeType:     p.volumeDecommission,
		gridtypes.NetworkType:    p.networkDecommission,
		gridtypes.ZDBType:        p.zdbDecommission,
		gridtypes.KubernetesType: p.kubernetesDecomission,
		gridtypes.PublicIPType:   p.publicIPDecomission,
	}

	p.Provisioner = provision.NewMapProvisioner(provisioners, decommissioners)

	return p
}

// RuntimeUpgrade runs upgrade needed when provision daemon starts
func (p *Primitives) RuntimeUpgrade(ctx context.Context) {
	p.upgradeRunningZdb(ctx)
}
