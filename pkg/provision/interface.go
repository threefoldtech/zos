package provision

import (
	"context"

	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
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

// Provisioner interface
type Provisioner interface {
	Provision(ctx context.Context, reservation *Reservation) (*Result, error)
	Decommission(ctx context.Context, reservation *Reservation) error
}

// ReservationConverterFunc is used to convert from the explorer workloads type into the
// internal Reservation type
type ReservationConverterFunc func(w workloads.Workloader) (*Reservation, error)

//ResultConverterFunc is used to convert internal Result type to the explorer workload result
type ResultConverterFunc func(result Result) (*workloads.Result, error)

// ReservationCache define the interface to store
// some reservations
type ReservationCache interface {
	Add(r *Reservation, override bool) error
	Get(id string) (*Reservation, error)
	Remove(id string) error
	Exists(id string) (bool, error)
	NetworkExists(id string) (bool, error)
}

// Statser is used by the provision Engine to keep
// track of how much resource unit and number of primitives
// is provisionned
type Statser interface {
	Increment(r *Reservation) error
	Decrement(r *Reservation) error
	CurrentUnits() directory.ResourceAmount
	CurrentWorkloads() directory.WorkloadAmount
	CheckMemoryRequirements(r *Reservation, totalMemAvailable uint64) error
}
