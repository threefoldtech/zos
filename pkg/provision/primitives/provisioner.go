package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type provisionFn func(ctx context.Context, wl *gridtypes.Workload) (interface{}, error)
type decommissionFn func(ctx context.Context, wl *gridtypes.Workload) error

// Primitives hold all the logic responsible to provision and decomission
// the different primitives workloads defined by this package
type Primitives struct {
	zbus zbus.Client

	provisioners    map[gridtypes.WorkloadType]provisionFn
	decommissioners map[gridtypes.WorkloadType]decommissionFn
}

var _ provision.Provisioner = (*Primitives)(nil)

// NewPrimitivesProvisioner creates a new 0-OS provisioner
func NewPrimitivesProvisioner(zbus zbus.Client) *Primitives {
	p := &Primitives{
		zbus: zbus,
	}
	p.provisioners = map[gridtypes.WorkloadType]provisionFn{
		gridtypes.ContainerType:  p.containerProvision,
		gridtypes.VolumeType:     p.volumeProvision,
		gridtypes.NetworkType:    p.networkProvision,
		gridtypes.ZDBType:        p.zdbProvision,
		gridtypes.KubernetesType: p.kubernetesProvision,
		gridtypes.PublicIPType:   p.publicIPProvision,
	}
	p.decommissioners = map[gridtypes.WorkloadType]decommissionFn{
		gridtypes.ContainerType:  p.containerDecommission,
		gridtypes.VolumeType:     p.volumeDecommission,
		gridtypes.NetworkType:    p.networkDecommission,
		gridtypes.ZDBType:        p.zdbDecommission,
		gridtypes.KubernetesType: p.kubernetesDecomission,
		gridtypes.PublicIPType:   p.publicIPDecomission,
	}

	return p
}

// RuntimeUpgrade runs upgrade needed when provision daemon starts
func (p *Primitives) RuntimeUpgrade(ctx context.Context) {
	p.upgradeRunningZdb(ctx)
}

// Provision implemenents provision.Provisioner
func (p *Primitives) Provision(ctx context.Context, wl *gridtypes.Workload) (*gridtypes.Result, error) {
	handler, ok := p.provisioners[wl.Type]
	if !ok {
		return nil, fmt.Errorf("unknown reservation type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	data, err := handler(ctx, wl)
	return p.buildResult(wl, data, err)
}

// Decommission implementation for provision.Provisioner
func (p *Primitives) Decommission(ctx context.Context, wl *gridtypes.Workload) error {
	handler, ok := p.decommissioners[wl.Type]
	if !ok {
		return fmt.Errorf("unknown reservation type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	return handler(ctx, wl)
}

func (p *Primitives) buildResult(wl *gridtypes.Workload, data interface{}, err error) (*gridtypes.Result, error) {
	result := &gridtypes.Result{
		Created: time.Now(),
	}

	if err != nil {
		result.Error = err.Error()
		result.State = gridtypes.StateError
	} else {
		result.State = gridtypes.StateOk
	}

	br, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	return result, nil
}
