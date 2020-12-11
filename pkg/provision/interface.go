package provision

import (
	"context"

	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg"
)

// ReservationSource interface. The source
// defines how the node will get reservation requests
// then reservations are applied to the node to deploy
// a resource of the given Reservation.Type
type ReservationSource interface {
	Reservations(ctx context.Context) <-chan *ReservationJob
}

// ReservationGetter interface. Some reservation sources
// can implement the getter interface
type ReservationGetter interface {
	Get(gwid string) (*Reservation, error)
}

// ReservationPoller define the interface to implement
// to poll the Explorer for new reservation
type ReservationPoller interface {
	ReservationGetter
	// Poll ask the store to send us reservation for a specific node ID
	// from is the used as a filter to which reservation to use as
	// reservation.ID >= from. So a client to the Poll method should make
	// sure to call it with the last (MAX) reservation ID he receieved.
	Poll(nodeID pkg.Identifier, from uint64) (reservations []*Reservation, lastID uint64, err error)
}

// Provisioner interface
type Provisioner interface {
	Provision(ctx context.Context, reservation *Reservation) (*Result, error)
	Decommission(ctx context.Context, reservation *Reservation) error
}

// Filter is filtering function for Purge method
type Filter func(*Reservation) bool

// ReservationCache define the interface to store
// some reservations
type ReservationCache interface {
	Add(r *Reservation, override bool) error
	Get(id string) (*Reservation, error)
	Remove(id string) error
	Exists(id string) (bool, error)
	Find(f Filter) ([]*Reservation, error)
}

// Counter is used by the provision Engine to keep
// track of how much resource unit and number of primitives
// is provisionned
type Counter interface {
	Increment(r *Reservation) error
	Decrement(r *Reservation) error
	CurrentUnits() directory.ResourceAmount
	CurrentWorkloads() directory.WorkloadAmount
	CheckMemoryRequirements(r *Reservation, totalMemAvailable uint64) error
}

// Janitor interface
type Janitor interface {
	Cleanup(ctx context.Context) error
}
