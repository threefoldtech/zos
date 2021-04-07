package provision

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Users is used to get user public key
type Users interface {
	GetKey(id gridtypes.ID) (ed25519.PublicKey, error)
}

// Engine is engine interface
type Engine interface {
	// Provision pushes a workload to engine queue. on success
	// means that workload has been committed to storage (accepts)
	// and will be processes later
	Provision(ctx context.Context, wl gridtypes.Deployment) error
	Deprovision(ctx context.Context, twin, id uint32, reason string) error
	Update(ctx context.Context, twin, id uint32, update gridtypes.Deployment) error
	Storage() Storage
	Users() Users
	Admins() Users
}

// Provisioner interface
type Provisioner interface {
	Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (*gridtypes.Result, error)
	Decommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error
}

// Filter is filtering function for Purge method

var (
	//ErrDeploymentExists returned if object exist
	ErrDeploymentExists = fmt.Errorf("exists")
	//ErrDeploymentNotExists returned if object not exists
	ErrDeploymentNotExists = fmt.Errorf("not exists")
)

// Storage interface
type Storage interface {
	Add(wl gridtypes.Deployment) error
	Set(wl gridtypes.Deployment) error
	Get(twin, deployment uint32) (gridtypes.Deployment, error)
	Twins() ([]uint32, error)
	ByTwin(twin uint32) ([]uint32, error)
}

// Janitor interface
type Janitor interface {
	Cleanup(ctx context.Context) error
}
