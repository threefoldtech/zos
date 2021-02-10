package provision

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"

	"github.com/rs/zerolog/log"
)

const gib = 1024 * 1024 * 1024

const minimunZosMemory = 2 * gib

// EngineOption interface
type EngineOption interface {
	apply(e *NativeEngine)
}

// WithJanitor sets a janitor for the engine.
// a janitor is executed periodically to clean up
// the deployed resources.
func WithJanitor(j Janitor) EngineOption {
	return &withJanitorOpt{j}
}

type provisionJob struct {
	wl gridtypes.Workload
	ch chan error
}

type deprovisionJob struct {
	id     gridtypes.ID
	ch     chan error
	reason string
}

// NativeEngine is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type NativeEngine struct {
	storage     Storage
	janitor     Janitor
	provisioner Provisioner

	provision   chan provisionJob
	deprovision chan deprovisionJob
}

var _ Engine = (*NativeEngine)(nil)
var _ pkg.Provision = (*NativeEngine)(nil)

type withJanitorOpt struct {
	j Janitor
}

func (o *withJanitorOpt) apply(e *NativeEngine) {
	panic("not implemented")
	e.janitor = o.j
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(storage Storage, provisioner Provisioner, opts ...EngineOption) *NativeEngine {
	e := &NativeEngine{
		storage:     storage,
		provisioner: provisioner,
		provision:   make(chan provisionJob),
		deprovision: make(chan deprovisionJob),
	}

	for _, opt := range opts {
		opt.apply(e)
	}

	return e
}

func (e *NativeEngine) Storage() Storage {
	return e.storage
}

// Provision workload
func (e *NativeEngine) Provision(ctx context.Context, wl gridtypes.Workload) error {
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
func (e *NativeEngine) Deprovision(ctx context.Context, id gridtypes.ID, reason string) error {
	j := deprovisionJob{
		id:     id,
		ch:     make(chan error),
		reason: reason,
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
func (e *NativeEngine) Run(ctx context.Context) error {
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

			e.uninstall(ctx, wl, job.reason)
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

func (e *NativeEngine) uninstall(ctx context.Context, wl gridtypes.Workload, reason string) {
	log := log.With().Str("id", string(wl.ID)).Str("type", string(wl.Type)).Logger()

	log.Debug().Msg("provisioning")
	if wl.Result.State == gridtypes.StateDeleted ||
		wl.Result.State == gridtypes.StateError {
		//nothing to do!
		return
	}

	err := e.provisioner.Decommission(ctx, &wl)
	result := &gridtypes.Result{
		Error: reason,
		State: gridtypes.StateDeleted,
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to deploy workload")
		result.State = gridtypes.StateError
		result.Error = errors.Wrapf(err, "error while decommission reservation because of: '%s'", result.Error).Error()
	}

	result.Created = time.Now()
	wl.Result = *result

	if err := e.storage.Set(wl); err != nil {
		log.Error().Err(err).Msg("failed to set workload result")
	}
}

func (e *NativeEngine) install(ctx context.Context, wl gridtypes.Workload) {
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

// Counters implements the zbus interface
func (e *NativeEngine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	//TODO: implement counters
	// this is probably need to be moved to
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			ch <- pkg.ProvisionCounters{}
		}
	}()

	return ch
}

// DecommissionCached implements the zbus interface
func (e *NativeEngine) DecommissionCached(id string, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	return e.Deprovision(ctx, gridtypes.ID(id), reason)
}
