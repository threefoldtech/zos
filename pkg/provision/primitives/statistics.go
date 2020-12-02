package primitives

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg/provision"
)

type (
	currentCapacityKey struct{}
)

// CurrentCapacity is a value that holds current system load in bytes
type CurrentCapacity struct {
	Cru uint64
	Mru uint64
	Hru uint64
	Sru uint64
}

// GetCapacity gets current capacity from context
func GetCapacity(ctx context.Context) CurrentCapacity {
	val := ctx.Value(currentCapacityKey{})
	if val == nil {
		panic("no current capacity injected")
	}

	return val.(CurrentCapacity)
}

// statsProvisioner a provisioner interceptor that keeps track
// of consumed capacity, and reprot it to the explorer
// when it has been changed
type statsProvisioner struct {
	inner    provision.Provisioner
	counters Counters
	reserved Counters

	nodeID string
	client client.Directory
}

// NewStatisticsProvisioner creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatisticsProvisioner(inner provision.Provisioner, initial, reserved Counters, nodeID string, client client.Directory) provision.Provisioner {
	return &statsProvisioner{inner: inner, counters: initial, reserved: reserved, nodeID: nodeID, client: client}
}

func (s *statsProvisioner) currentCapacity() CurrentCapacity {
	return CurrentCapacity{
		Cru: s.counters.CRU.Current() + s.reserved.CRU.Current(),
		Mru: s.counters.CRU.Current() + s.reserved.MRU.Current(),
		Hru: s.counters.CRU.Current() + s.reserved.HRU.Current(),
		Sru: s.counters.CRU.Current() + s.reserved.SRU.Current(),
	}
}

func (s *statsProvisioner) Provision(ctx context.Context, reservation *provision.Reservation) (*provision.Result, error) {
	current := s.currentCapacity()
	ctx = context.WithValue(ctx, currentCapacityKey{}, current)
	result, err := s.inner.Provision(ctx, reservation)
	if err != nil {
		return result, err
	}

	if err := s.counters.Increment(reservation); err != nil {
		log.Error().Err(err).Msg("failed to increment statistics counter")
	}

	s.sync(reservation.NodeID)

	return result, nil
}

func (s *statsProvisioner) Decommission(ctx context.Context, reservation *provision.Reservation) error {
	if err := s.inner.Decommission(ctx, reservation); err != nil {
		return err
	}

	if err := s.counters.Decrement(reservation); err != nil {
		log.Error().Err(err).Msg("failed to decrement statistics counter")
	}

	s.sync(reservation.NodeID)

	return nil
}

func (s *statsProvisioner) sync(nodeID string) {
	strategy := backoff.NewExponentialBackOff()
	strategy.MaxInterval = 15 * time.Second
	strategy.MaxElapsedTime = 3 * time.Minute

	current := s.counters.CurrentUnits()
	reserved := s.reserved.CurrentUnits()

	current.Cru += reserved.Cru
	current.Sru += reserved.Sru
	current.Hru += reserved.Hru
	current.Mru += reserved.Mru

	err := backoff.Retry(func() error {
		err := s.client.NodeUpdateUsedResources(
			s.nodeID,
			current,
			s.counters.CurrentWorkloads(),
		)

		if err == nil || errors.Is(err, client.ErrRequestFailure) {
			// we only retry if err is a request failure err.
			return err
		}

		// otherwise retrying won't fix it, so we can terminate
		return backoff.Permanent(err)
	}, strategy)

	if err != nil {
		log.Error().Err(err).Msg("failed to update consumed capacity on explorer")
	}
}
