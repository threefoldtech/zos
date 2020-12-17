package provision

import (
	"bytes"
	"context"
	"fmt"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type (
	cacheKey struct{}
)

// GetCache from context
func GetCache(ctx context.Context) ReservationCache {
	c := ctx.Value(cacheKey{})
	if c == nil {
		panic("cache middleware is not set")
	}

	return c.(ReservationCache)
}

type cachedProvisioner struct {
	inner Provisioner
	cache ReservationCache

	mem *cache.Cache
}

// NewCachedProvisioner is a provisioner interceptor that checks a cache to avoid
// rerunning the reservation logic.
// on deprovision it skips if there is no cached reservation
func NewCachedProvisioner(c ReservationCache, inner Provisioner) Provisioner {
	return &cachedProvisioner{
		inner: inner,
		cache: c,
	}
}

func (c *cachedProvisioner) Provision(ctx context.Context, reservation *Reservation) (*Result, error) {
	if len(reservation.Reference) != 0 {
		result, err := c.migrateToPool(ctx, reservation)
		if err != nil || result != nil {
			return result, err
		}
		// otherwise we need to provision this reservation
	}

	if cached, err := c.cache.Get(reservation.ID); err == nil {
		log.Info().Str("id", reservation.ID).Msg("reservation have already been processed")
		if cached.Result.IsNil() {
			// this is probably an older reservation that is cached BEFORE
			// we start caching the result along with the reservation
			// then we just need to return here.
			return nil, nil
		}

		return &cached.Result, nil
	}

	id := reservation.ID
	if len(reservation.Reference) != 0 {
		reservation.ID = reservation.Reference
	}

	ctx = context.WithValue(ctx, cacheKey{}, c.cache)
	result, err := c.inner.Provision(
		ctx,
		reservation,
	)

	if err != nil {
		return result, err
	}

	if len(reservation.Reference) != 0 {
		result.ID = id
		reservation.ID = id
	}

	// we only cache successfull reservations
	if result.State == StateOk {
		reservation.Result = *result
		if err := c.cache.Add(reservation, false); err != nil {
			return result, errors.Wrap(err, "cache: failed to cache reservation result")
		}
	}

	return result, nil
}

func (c *cachedProvisioner) Decommission(ctx context.Context, reservation *Reservation) error {
	exists, err := c.cache.Exists(reservation.ID)
	if err != nil {
		return errors.Wrap(err, "cache: failed to check if reservation already exists")
	}

	if !exists {
		return nil
	}

	if err := c.inner.Decommission(ctx, reservation); err != nil {
		return err
	}

	return c.cache.Remove(reservation.ID)
}

func (c *cachedProvisioner) Get(ctx context.Context, id string) (*Reservation, error) {
	reservation, err := c.cache.Get(id)
	if err != nil {
		return nil, errors.Wrapf(ErrUnknownReservation, "reservation not cached: %s", err)
	}

	return reservation, nil
}

func (c *cachedProvisioner) migrateToPool(ctx context.Context, r *Reservation) (*Result, error) {
	oldRes, err := c.cache.Get(r.Reference)
	if err != nil {
		// not cached. so nothing to do
		return nil, nil
	}

	log := log.With().Str("reference", r.Reference).Str("id", r.ID).Logger()

	// we have received a reservation that reference another one.
	// This is the sign user is trying to migrate his workloads to the new capacity pool system

	log.Info().Msg("reservation referencing another one")

	if string(oldRes.Type) != "network" { //we skip network cause its a PITA
		// first let make sure both are the same
		if !bytes.Equal(oldRes.Data, r.Data) {
			return nil, fmt.Errorf("trying to upgrade workloads to new version. new workload content is different from the old one. upgrade refused")
		}
	}

	// remove the old one from the cache and store the new one
	log.Info().Msg("migration: remove from cache")
	if err := c.cache.Remove(oldRes.ID); err != nil {
		return nil, err
	}

	r.Result = oldRes.Result
	r.Result.ID = r.ID

	log.Info().Msg("migration: add to cache")
	if err := c.cache.Add(r, true); err != nil {
		return nil, err
	}

	return &r.Result, nil
}
