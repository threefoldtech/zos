package primitives

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg/provision"
)

type (
	currentCapacity struct{}
)

// GetCapacity gets current capacity from context
func GetCapacity(ctx context.Context) directory.ResourceAmount {
	val := ctx.Value(currentCapacity{})
	if val == nil {
		panic("no current capacity injected")
	}

	return val.(directory.ResourceAmount)
}

// statsProvisioner a provisioner interceptor that keeps track
// of consumed capacity, and reprot it to the explorer
// when it has been changed
type statsProvisioner struct {
	inner    provision.Provisioner
	counters Counters

	nodeID string
	client client.Directory
}

// NewStatisticsProvisioner creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatisticsProvisioner(inner provision.Provisioner, initial Counters, nodeID string, client client.Directory) provision.Provisioner {
	return &statsProvisioner{inner: inner, counters: initial, nodeID: nodeID, client: client}
}

func (s *statsProvisioner) Provision(ctx context.Context, reservation *provision.Reservation) (*provision.Result, error) {
	current := s.counters.CurrentUnits()
	ctx = context.WithValue(ctx, currentCapacity{}, current)
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

func (s *statsProvisioner) sync(nodeID string) {
	strategy := backoff.NewExponentialBackOff()
	strategy.MaxInterval = 15 * time.Second
	strategy.MaxElapsedTime = 3 * time.Minute

	// TODO (VERY IMPORTANT)
	// make sure to also report used capacity
	// by the node itself, such as cache volume
	err := backoff.Retry(func() error {
		err := s.client.NodeUpdateUsedResources(
			s.nodeID,
			s.counters.CurrentUnits(),
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
