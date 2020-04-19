// Package provision is a daemon that pulls
// on reservation source, and then tries to
// apply these reservations locally.
// Note that, provision module doesn't expose
// any interface on zbus. since it should not
// be driven by users, instead all reservation
// should be pushed by the reservation source.
package provision

import (
	"context"

	"github.com/threefoldtech/zbus"
)

// ReservationSource interface. The source
// defines how the node will get reservation requests
// then reservations are applied to the node to deploy
// a resource of the given Reservation.Type
type ReservationSource interface {
	Reservations(ctx context.Context) <-chan *Reservation
}

type ProvisionerFunc func(ctx context.Context, reservation *Reservation) (interface{}, error)
type DecommissionerFunc func(ctx context.Context, reservation *Reservation) error

type Provisioner struct {
	cache OwnerCache
	zbus  zbus.Client

	Provisioners    map[ReservationType]ProvisionerFunc
	Decommissioners map[ReservationType]DecommissionerFunc
}

func NewProvisioner(owerCache OwnerCache, zbus zbus.Client) *Provisioner {
	p := &Provisioner{
		cache: owerCache,
		zbus:  zbus,
	}
	p.Provisioners = map[ReservationType]ProvisionerFunc{
		ContainerReservation:  p.containerProvision,
		VolumeReservation:     p.volumeProvision,
		NetworkReservation:    p.networkProvision,
		ZDBReservation:        p.zdbProvision,
		DebugReservation:      p.debugProvision,
		KubernetesReservation: p.kubernetesProvision,
	}
	p.Decommissioners = map[ReservationType]DecommissionerFunc{
		ContainerReservation:  p.containerDecommission,
		VolumeReservation:     p.volumeDecommission,
		NetworkReservation:    p.networkDecommission,
		ZDBReservation:        p.zdbDecommission,
		DebugReservation:      p.debugDecommission,
		KubernetesReservation: p.kubernetesDecomission,
	}

	return p
}

var (
// provisioners defines the entry point for the different
// reservation provisioners. Currently only containers are
// supported.

)
