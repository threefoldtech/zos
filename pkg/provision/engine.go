package provision

import (
	"context"

	"github.com/threefoldtech/zos/pkg/gridtypes"

	"github.com/rs/zerolog/log"
)

const gib = 1024 * 1024 * 1024

const minimunZosMemory = 2 * gib

// EngineImpl is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type EngineImpl struct {
	provision   chan gridtypes.Workload
	deprovision chan gridtypes.ID
	provisioner Provisioner
	janitor     Janitor
}

// EngineOps are the configuration of the engine
type EngineOps struct {
	Provisioner Provisioner

	// Janitor is used to clean up some of the resources that might be lingering on the node
	// if not set, no cleaning up will be done
	Janitor Janitor
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(opts EngineOps) *EngineImpl {
	return &EngineImpl{
		provision:   make(chan gridtypes.Workload),
		deprovision: make(chan gridtypes.ID),
		provisioner: opts.Provisioner,
		janitor:     opts.Janitor,
	}
}

// Run starts reader reservation from the Source and handle them
func (e *EngineImpl) Run(ctx context.Context) error {
	defer close(e.provision)
	defer close(e.deprovision)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _ = <-e.deprovision:
			//e.provisioner.Decommission(ctx context.Context, wl *gridtypes.Workload)
		case wl := <-e.provision:
			log := log.With().Str("id", string(wl.ID)).Str("type", string(wl.Type)).Logger()
			//TODO:
			//1- commit to storage
			//2- apply
			log.Debug().Msg("provisioning")
			e.provisioner.Provision(ctx, &wl)
		}
	}
}
