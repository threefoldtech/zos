package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Manager defines basic type manager functionality. This interface
// declares the provision and the deprovision method which is required
// by any Type manager.
type Manager interface {
	Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error)
	Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error
}

// Initializer interface define an extra Initialize method which is run on the provisioner
// before the provision engine is started.
type Initializer interface {
	Initialize(ctx context.Context) error
}

// Updater defines the optional Update method for a type manager. Types are allowed
// to implement update to change their settings on the fly
type Updater interface {
	Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error)
}

// Pauser defines optional Pause, Resume method for type managers. Types are allowed
// to implement pause, resume to put the workload in paused state where it's not usable
// by the user but at the same time not completely deleted.
type Pauser interface {
	Pause(ctx context.Context, wl *gridtypes.WorkloadWithID) error
	Resume(ctx context.Context, wl *gridtypes.WorkloadWithID) error
}

type mapProvisioner struct {
	managers map[gridtypes.WorkloadType]Manager
}

// NewMapProvisioner returns a new instance of a map provisioner
func NewMapProvisioner(managers map[gridtypes.WorkloadType]Manager) Provisioner {
	return &mapProvisioner{
		managers: managers,
	}
}

func (p *mapProvisioner) Initialize(ctx context.Context) error {
	for typ, mgr := range p.managers {
		init, ok := mgr.(Initializer)
		if !ok {
			continue
		}

		if err := init.Initialize(ctx); err != nil {
			return errors.Wrapf(err, "failed to run initializers for workload type '%s'", typ)
		}
	}

	return nil
}

// Provision implements provision.Provisioner
func (p *mapProvisioner) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (result gridtypes.Result, err error) {
	manager, ok := p.managers[wl.Type]
	if !ok {
		return result, fmt.Errorf("unknown workload type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	data, err := manager.Provision(ctx, wl)
	if errors.Is(err, ErrNoActionNeeded) {
		return result, err
	}

	return p.buildResult(data, err)
}

// Decommission implementation for provision.Provisioner
func (p *mapProvisioner) Decommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	manager, ok := p.managers[wl.Type]
	if !ok {
		return fmt.Errorf("unknown workload type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	return manager.Deprovision(ctx, wl)
}

// Provision implements provision.Provisioner
func (p *mapProvisioner) Update(ctx context.Context, wl *gridtypes.WorkloadWithID) (result gridtypes.Result, err error) {
	manager, ok := p.managers[wl.Type]
	if !ok {
		return result, fmt.Errorf("unknown workload type '%s' for reservation id '%s'", wl.Type, wl.ID)
	}

	updater, ok := manager.(Updater)
	if !ok {
		return result, fmt.Errorf("workload type '%s' does not support updating", wl.Type)
	}

	data, err := updater.Update(ctx, wl)
	if errors.Is(err, ErrNoActionNeeded) {
		return result, err
	}

	return p.buildResult(data, err)
}

func (p *mapProvisioner) CanUpdate(ctx context.Context, typ gridtypes.WorkloadType) bool {
	manager, ok := p.managers[typ]
	if !ok {
		return false
	}

	_, ok = manager.(Updater)
	return ok
}

func (p *mapProvisioner) buildResult(data interface{}, err error) (gridtypes.Result, error) {
	result := gridtypes.Result{
		Created: gridtypes.Timestamp(time.Now().Unix()),
	}

	var unchanged ErrUnchanged
	if errors.As(err, &unchanged) {
		result.Error = unchanged.Error()
		result.State = gridtypes.StateUnChanged
	} else if err != nil {
		result.Error = err.Error()
		result.State = gridtypes.StateError
	} else {
		result.State = gridtypes.StateOk
	}

	br, err := json.Marshal(data)
	if err != nil {
		return result, errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	return result, nil
}
