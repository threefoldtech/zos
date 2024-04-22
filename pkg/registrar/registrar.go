package registrar

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// should any of this be moved to pkg?
type RegistrationState string

const (
	Failed     RegistrationState = "Failed"
	InProgress RegistrationState = "InProgress"
	Done       RegistrationState = "Done"

	monitorAccountEvery = 30 * time.Minute
)

var (
	ErrInProgress = errors.New("registration is still in progress")
	ErrFailed     = errors.New("registration failed")
)

type State struct {
	NodeID uint32
	TwinID uint32
	State  RegistrationState
	Msg    string
}

func FailedState(err error) State {
	return State{
		0,
		0,
		Failed,
		err.Error(),
	}
}

func InProgressState() State {
	return State{
		0,
		0,
		InProgress,
		"",
	}
}

func DoneState(nodeID uint32, twinID uint32) State {
	return State{
		nodeID,
		twinID,
		Done,
		"",
	}
}

type Registrar struct {
	state State
	mutex sync.RWMutex
}

func NewRegistrar(ctx context.Context, cl zbus.Client, env environment.Environment, info RegistrationInfo) *Registrar {
	r := Registrar{
		State{
			0,
			0,
			InProgress,
			"",
		},
		sync.RWMutex{},
	}
	go r.register(ctx, cl, env, info)
	return &r
}

func (r *Registrar) setState(s State) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.state = s
}

func (r *Registrar) getState() State {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.state
}

// register a node and then blocks forever watching the node account. It tries to re-activate the
// account if needed
func (r *Registrar) register(ctx context.Context, cl zbus.Client, env environment.Environment, info RegistrationInfo) {
	if app.CheckFlag(app.LimitedCache) {
		r.setState(FailedState(errors.New("no disks")))
		return
	}
	if _, err := os.Stat("/dev/kvm"); err != nil {
		r.setState(FailedState(errors.New("virtualization is not enabled. please enable in BIOS")))
		return
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Minute
	exp.MaxElapsedTime = 0 // retry indefinitely
	bo := backoff.WithContext(exp, ctx)
	register := func() {
		err := backoff.RetryNotify(func() error {
			nodeID, twinID, err := r.registration(ctx, cl, env, info)
			if err != nil {
				r.setState(FailedState(err))
				return err
			} else {
				r.setState(DoneState(nodeID, twinID))
			}
			return nil
		}, bo, retryNotify)
		if err != nil {
			// this should never happen because we retry indefinitely
			log.Error().Err(err).Msg("registration failed")
			return
		}
	}

	register()

	stub := stubs.NewNetworkerStub(cl)
	addressesUpdate, err := stub.ZOSAddresses(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to monitor ip changes")
		return
	}

	for {
		select {
		case <-ctx.Done():
		case <-time.After(monitorAccountEvery):
			if err := r.reActivate(ctx, cl, env); err != nil {
				log.Error().Err(err).Msg("failed to reactivate account")
			}
		case <-addressesUpdate:
			register()
		}
	}
}

func (r *Registrar) reActivate(ctx context.Context, cl zbus.Client, env environment.Environment) error {
	apiGateway := stubs.NewAPIGatewayStub(cl)

	_, err := apiGateway.EnsureAccount(ctx, env.ActivationURL, tcUrl, tcHash)

	return err
}

func (r *Registrar) NodeID() (uint32, error) {
	return r.returnIfDone(r.getState().NodeID)
}

func (r *Registrar) TwinID() (uint32, error) {
	return r.returnIfDone(r.getState().TwinID)
}

func (r *Registrar) returnIfDone(v uint32) (uint32, error) {
	if r.state.State == Failed {
		return 0, errors.Wrap(ErrFailed, r.state.Msg)
	} else if r.state.State == Done {
		return v, nil
	} else {
		// InProgress
		return 0, ErrInProgress
	}
}
