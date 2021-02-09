package provision

import (
	"context"
	"time"

	"github.com/threefoldtech/zos/pkg/gridtypes"

	"github.com/rs/zerolog/log"
)

const gib = 1024 * 1024 * 1024

const minimunZosMemory = 2 * gib

type provisionJob struct {
	wl gridtypes.Workload
	ch chan error
}

type deprovisionJob struct {
	id gridtypes.ID
	ch chan error
}

// EngineImpl is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type EngineImpl struct {
	storage     Storage
	janitor     Janitor
	provisioner Provisioner

	provision   chan provisionJob
	deprovision chan deprovisionJob
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
		provision:   make(chan provisionJob),
		deprovision: make(chan deprovisionJob),
		provisioner: opts.Provisioner,
		janitor:     opts.Janitor,
	}
}

// Provision workload
func (e *EngineImpl) Provision(ctx context.Context, wl gridtypes.Workload) error {
	j := provisionJob{
		wl: wl,
		ch: make(chan error),
	}

	defer close(j.ch)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.provision <- j:
		return <-j.ch
	}
}

// Deprovision workload
func (e *EngineImpl) Deprovision(ctx context.Context, id gridtypes.ID) error {
	j := deprovisionJob{
		id: id,
		ch: make(chan error),
	}

	defer close(j.ch)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e.deprovision <- j:
		return <-j.ch
	}
}

// Run starts reader reservation from the Source and handle them
func (e *EngineImpl) Run(ctx context.Context) error {
	defer close(e.provision)
	defer close(e.deprovision)

	ctx = context.WithValue(ctx, storageKey{}, e.storage)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case job := <-e.deprovision:
			wl, err := e.storage.Get(job.id)
			if err != nil {
				job.ch <- err
				log.Error().Err(err).Stringer("id", job.id).Msg("failed to get workload from storage")
				continue
			}
			wl.ToDelete = true
			err = e.storage.Set(wl)
			job.ch <- err
			if err != nil {
				log.Error().Err(err).Stringer("id", job.id).Msg("failed to mark workload at to be delete")
				continue
			}

			e.uninstall(ctx, wl)
		case job := <-e.provision:
			job.wl.Created = time.Now()
			job.wl.ToDelete = false
			err := e.storage.Add(job.wl)
			// release the job. the caller will now know that the workload
			// has been committed to storage (or not)
			job.ch <- err
			if err != nil {
				log.Error().Err(err).Stringer("id", job.wl.ID).Msg("failed to commit workload to storage")
				continue
			}

			e.install(ctx, job.wl)
		}
	}
}

func (e *EngineImpl) uninstall(ctx context.Context, wl gridtypes.Workload) {
	log := log.With().Str("id", string(wl.ID)).Str("type", string(wl.Type)).Logger()

	log.Debug().Msg("provisioning")
	result, err := e.provisioner.Provision(ctx, &wl)
	if err != nil {
		log.Error().Err(err).Msg("failed to deploy workload")
		result = &gridtypes.Result{
			Error: err.Error(),
			State: gridtypes.StateError,
		}
	}

	result.Created = time.Now()
	wl.Result = *result

	if err := e.storage.Set(wl); err != nil {
		log.Error().Err(err).Msg("failed to set workload result")
	}
}

func (e *EngineImpl) install(ctx context.Context, wl gridtypes.Workload) {
	log := log.With().Str("id", string(wl.ID)).Str("type", string(wl.Type)).Logger()

	log.Debug().Msg("provisioning")
	result, err := e.provisioner.Provision(ctx, &wl)
	if err != nil {
		log.Error().Err(err).Msg("failed to deploy workload")
		result = &gridtypes.Result{
			Error: err.Error(),
			State: gridtypes.StateError,
		}
	}

	result.Created = time.Now()
	wl.Result = *result

	if err := e.storage.Set(wl); err != nil {
		log.Error().Err(err).Msg("failed to set workload result")
	}
}
