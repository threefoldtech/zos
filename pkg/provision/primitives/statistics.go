package primitives

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
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
}

// NewStatisticsProvisioner creates a new statistics provisioner interceptor.
// Statistics provisioner keeps track of used capacity and update explorer when it changes
func NewStatisticsProvisioner(initial, reserved Counters, nodeID string, inner provision.Provisioner) provision.Provisioner {
	return &statsProvisioner{inner: inner, counters: initial, reserved: reserved, nodeID: nodeID}
}

func (s *statsProvisioner) currentCapacity() CurrentCapacity {
	return CurrentCapacity{
		Cru: s.counters.CRU.Current() + s.reserved.CRU.Current(),
		Mru: s.counters.CRU.Current() + s.reserved.MRU.Current(),
		Hru: s.counters.CRU.Current() + s.reserved.HRU.Current(),
		Sru: s.counters.CRU.Current() + s.reserved.SRU.Current(),
	}
}

func (s *statsProvisioner) Provision(ctx context.Context, wl *gridtypes.Workload) (*gridtypes.Result, error) {
	current := s.currentCapacity()
	ctx = context.WithValue(ctx, currentCapacityKey{}, current)
	result, err := s.inner.Provision(ctx, wl)
	if err != nil {
		return result, err
	}

	if result.State == gridtypes.StateOk {
		if err := s.counters.Increment(wl); err != nil {
			log.Error().Err(err).Msg("failed to increment statistics counter")
		}
	}

	return result, nil
}

func (s *statsProvisioner) Decommission(ctx context.Context, wl *gridtypes.Workload) error {
	if err := s.inner.Decommission(ctx, wl); err != nil {
		return err
	}

	if err := s.counters.Decrement(wl); err != nil {
		log.Error().Err(err).Msg("failed to decrement statistics counter")
	}

	//s.sync(wl.NodeID)

	return nil
}

func (s *statsProvisioner) shouldUpdateCounters(ctx context.Context, wl *gridtypes.Workload) (bool, error) {
	// rule, we always should update counters UNLESS it is a network reservation that
	// already have been counted before.
	if wl.Type != gridtypes.NetworkReservation {
		return true, nil
	}

	var nr gridtypes.Network
	if err := json.Unmarshal(wl.Data, &nr); err != nil {
		return false, fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}
	// otherwise we check the cache if a network
	// with the same id already exists
	id := gridtypes.NetworkID(wl.User.String(), nr.Name)
	cache := provision.GetStorage(ctx)
	_, err := cache.GetNetwork(id)
	if errors.Is(err, provision.ErrWorkloadNotExists) {
		return true, nil
	} else if err != nil {
		return false, err
	}

	return false, nil
}
