package primitives

import (
	"context"
	"fmt"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Primitives hold all the logic responsible to provision and decomission
// the different primitives workloads defined by this package
type Primitives struct {
	cache provision.ReservationCache
	zbus  zbus.Client

	provisioners    map[provision.ReservationType]provision.ProvisionerFunc
	decommissioners map[provision.ReservationType]provision.DecomissionerFunc
}

// NewPrimitivesProvisioner creates a new 0-OS provisioner
func NewPrimitivesProvisioner(cache provision.ReservationCache, zbus zbus.Client) *Primitives {
	p := &Primitives{
		cache: cache,
		zbus:  zbus,
	}
	p.provisioners = map[provision.ReservationType]provision.ProvisionerFunc{
		ContainerReservation:       p.containerProvision,
		VolumeReservation:          p.volumeProvision,
		NetworkReservation:         p.networkProvision,
		NetworkResourceReservation: p.networkProvision,
		ZDBReservation:             p.zdbProvision,
		DebugReservation:           p.debugProvision,
		KubernetesReservation:      p.kubernetesProvision,
		PublicIPReservation:        p.publicIPProvision,
	}
	p.decommissioners = map[provision.ReservationType]provision.DecomissionerFunc{
		ContainerReservation:       p.containerDecommission,
		VolumeReservation:          p.volumeDecommission,
		NetworkReservation:         p.networkDecommission,
		NetworkResourceReservation: p.networkDecommission,
		ZDBReservation:             p.zdbDecommission,
		DebugReservation:           p.debugDecommission,
		KubernetesReservation:      p.kubernetesDecomission,
		PublicIPReservation:        p.publicIPDecomission,
	}

	return p
}

// RuntimeUpgrade runs upgrade needed when provision daemon starts
func (p *Primitives) RuntimeUpgrade(ctx context.Context) {
	p.upgradeRunningZdb(ctx)
}

// Provision implemenents provision.Provisioner
func (p *Primitives) Provision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	handler, ok := p.provisioners[reservation.Type]
	if !ok {
		return nil, fmt.Errorf("unknown reservation type '%s' for reservation id '%s'", reservation.Type, reservation.ID)
	}

	return handler(ctx, reservation)
}

// Decommission implementation for provision.Provisioner
func (p *Primitives) Decommission(ctx context.Context, reservation *provision.Reservation) error {
	handler, ok := p.decommissioners[reservation.Type]
	if !ok {
		return fmt.Errorf("unknown reservation type '%s' for reservation id '%s'", reservation.Type, reservation.ID)
	}

	return handler(ctx, reservation)
}
