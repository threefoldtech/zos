package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

type provisionFn func(ctx context.Context, reservation *provision.Reservation) (interface{}, error)
type decommissionFn func(ctx context.Context, reservation *provision.Reservation) error

// Primitives hold all the logic responsible to provision and decomission
// the different primitives workloads defined by this package
type Primitives struct {
	zbus zbus.Client

	provisioners    map[provision.ReservationType]provisionFn
	decommissioners map[provision.ReservationType]decommissionFn
}

var _ provision.Provisioner = (*Primitives)(nil)

// NewPrimitivesProvisioner creates a new 0-OS provisioner
func NewPrimitivesProvisioner(zbus zbus.Client) *Primitives {
	p := &Primitives{
		zbus: zbus,
	}
	p.provisioners = map[provision.ReservationType]provisionFn{
		ContainerReservation:       p.containerProvision,
		VolumeReservation:          p.volumeProvision,
		NetworkReservation:         p.networkProvision,
		NetworkResourceReservation: p.networkProvision,
		ZDBReservation:             p.zdbProvision,
		DebugReservation:           p.debugProvision,
		KubernetesReservation:      p.kubernetesProvision,
		PublicIPReservation:        p.publicIPProvision,
	}
	p.decommissioners = map[provision.ReservationType]decommissionFn{
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

// Get implements provision.Provisioner interface
func (p *Primitives) Get(ctx context.Context, id string) (*provision.Reservation, error) {
	return nil, provision.ErrUnknownReservation
}

// Provision implemenents provision.Provisioner
func (p *Primitives) Provision(ctx context.Context, reservation *provision.Reservation) (*provision.Result, error) {
	handler, ok := p.provisioners[reservation.Type]
	if !ok {
		return nil, fmt.Errorf("unknown reservation type '%s' for reservation id '%s'", reservation.Type, reservation.ID)
	}

	data, err := handler(ctx, reservation)
	return p.buildResult(reservation, data, err)
}

// Decommission implementation for provision.Provisioner
func (p *Primitives) Decommission(ctx context.Context, reservation *provision.Reservation) error {
	handler, ok := p.decommissioners[reservation.Type]
	if !ok {
		return fmt.Errorf("unknown reservation type '%s' for reservation id '%s'", reservation.Type, reservation.ID)
	}

	return handler(ctx, reservation)
}

func (p *Primitives) buildResult(reservation *provision.Reservation, data interface{}, err error) (*provision.Result, error) {
	result := &provision.Result{
		Type:    reservation.Type,
		Created: time.Now(),
		ID:      reservation.ID,
	}

	if err != nil {
		result.Error = err.Error()
		result.State = provision.StateError
	} else {
		result.State = provision.StateOk
	}

	br, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	return result, nil
}
