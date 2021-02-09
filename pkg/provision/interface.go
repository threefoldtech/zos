package provision

import (
	"context"
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Engine is engine interface
type Engine interface {
	Provision() chan<- gridtypes.Workload
	Deprovision() chan<- gridtypes.ID
	Get(gridtypes.ID) (gridtypes.Workload, error)
}

// Provisioner interface
type Provisioner interface {
	Provision(ctx context.Context, wl *gridtypes.Workload) (*gridtypes.Result, error)
	Decommission(ctx context.Context, wl *gridtypes.Workload) error
}

// Filter is filtering function for Purge method
type Filter func(*Reservation) bool

var (
	//ErrWorkloadExists returned if object exist
	ErrWorkloadExists = fmt.Errorf("exists")
	//ErrWorkloadNotExists returned if object not exists
	ErrWorkloadNotExists = fmt.Errorf("not exists")
)

// Storage interface
type Storage interface {
	Add(wl gridtypes.Workload) error
	Set(wl gridtypes.Workload) error
	Get(id gridtypes.ID) (gridtypes.Workload, error)

	// listing
	ByType(t gridtypes.ReservationType) ([]gridtypes.ID, error)
	ByUser(user gridtypes.ID, t gridtypes.ReservationType) ([]gridtypes.ID, error)
	Network(id gridtypes.NetID) error
}

// Janitor interface
type Janitor interface {
	Cleanup(ctx context.Context) error
}
