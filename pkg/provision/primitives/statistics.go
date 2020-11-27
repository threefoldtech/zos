package primitives

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg/provision"
)

var (
	currentCapacity = struct{}{}
)

// GetCapacity gets current capacity from context
func GetCapacity(ctx context.Context) directory.ResourceAmount {
	val := ctx.Value(currentCapacity)
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

	//todo: add explorer client here
}

// NewStatisticsProvisioner creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatisticsProvisioner(inner provision.Provisioner, initial Counters) provision.Provisioner {
	return &statsProvisioner{inner: inner, counters: initial}
}

func (s *statsProvisioner) Provision(ctx context.Context, reservation *provision.Reservation) (*provision.Result, error) {
	current := s.counters.CurrentUnits()
	ctx = context.WithValue(ctx, currentCapacity, current)
	result, err := s.inner.Provision(ctx, reservation)
	if err != nil {
		return result, err
	}

	if err := s.counters.Increment(reservation); err != nil {
		log.Error().Err(err).Msg("failed to increment statistics counter")
	}

	//TODO: send updates to explorer

	return result, nil
}

func (s *statsProvisioner) Decommission(ctx context.Context, reservation *provision.Reservation) error {
	if err := s.inner.Decommission(ctx, reservation); err != nil {
		return err
	}

	if err := s.counters.Decrement(reservation); err != nil {
		log.Error().Err(err).Msg("failed to decrement statistics counter")
	}

	return nil
}
