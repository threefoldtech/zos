package provision

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ReservationCache define the interface to store
// some reservations
type ReservationCache interface {
	Add(r *Reservation) error
	Get(id string) (*Reservation, error)
	Remove(id string) error
	Exists(id string) (bool, error)
	Counters() Counters
}

// Feedbacker defines the method that needs to be implemented
// to send the provision result to BCDB
type Feedbacker interface {
	Feedback(nodeID string, r *Result) error
	Deleted(nodeID, id string) error
	UpdateReservedResources(nodeID string, c Counters) error
}

type Signer interface {
	Sign(b []byte) ([]byte, error)
}

type Engine struct {
	nodeID   string
	source   ReservationSource
	cache    ReservationCache
	feedback Feedbacker
	// cl             *client.Client
	provisioners   map[ReservationType]ProvisionerFunc
	decomissioners map[ReservationType]DecommissionerFunc
	signer         Signer
}

type EngineOps struct {
	NodeID         string
	Source         ReservationSource
	Cache          ReservationCache
	Feedback       Feedbacker
	Provisioners   map[ReservationType]ProvisionerFunc
	Decomissioners map[ReservationType]DecommissionerFunc
	Signer         Signer
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(opts EngineOps) *Engine {
	return &Engine{
		nodeID:         opts.NodeID,
		source:         opts.Source,
		cache:          opts.Cache,
		feedback:       opts.Feedback,
		decomissioners: opts.Decomissioners,
		signer:         opts.Signer,
	}
}

// Run starts processing reservation resource. Then try to allocate
// reservations
func (e *Engine) Run(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cReservation := e.source.Reservations(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("provision engine context done, exiting")
			return nil

		case reservation, ok := <-cReservation:
			if !ok {
				log.Info().Msg("reservation source is emptied. stopping engine")
				return nil
			}

			expired := reservation.Expired()
			slog := log.With().
				Str("id", string(reservation.ID)).
				Str("type", string(reservation.Type)).
				Str("duration", fmt.Sprintf("%v", reservation.Duration)).
				Str("tag", reservation.Tag.String()).
				Bool("to-delete", reservation.ToDelete).
				Bool("expired", expired).
				Logger()

			if expired || reservation.ToDelete {
				slog.Info().Msg("start decommissioning reservation")
				if err := e.decommission(ctx, reservation); err != nil {
					log.Error().Err(err).Msgf("failed to decommission reservation %s", reservation.ID)
					continue
				}
			} else {
				slog.Info().Msg("start provisioning reservation")
				if err := e.provision(ctx, reservation); err != nil {
					log.Error().Err(err).Msgf("failed to provision reservation %s", reservation.ID)
					continue
				}
			}

			if err := e.feedback.UpdateReservedResources(e.nodeID, e.cache.Counters()); err != nil {
				log.Error().Err(err).Msg("failed to updated the capacity counters")
			}

		}
	}
}

func (e *Engine) provision(ctx context.Context, r *Reservation) error {
	if err := r.validate(); err != nil {
		return errors.Wrapf(err, "failed validation of reservation")
	}

	fn, ok := e.provisioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	_, err := e.cache.Get(r.ID)
	if err == nil {
		log.Info().Str("id", r.ID).Msg("reservation already deployed")
		return nil
	}

	result, err := fn(ctx, r)
	if err != nil {
		log.Error().
			Err(err).
			Str("id", r.ID).
			Msgf("failed to apply provision")
	} else {
		log.Info().
			Str("result", fmt.Sprintf("%v", result)).
			Msgf("workload deployed")
	}

	if replyErr := e.reply(ctx, r, err, result); replyErr != nil {
		log.Error().Err(replyErr).Msg("failed to send result to BCDB")
	}

	if err != nil {
		return err
	}

	if err := e.cache.Add(r); err != nil {
		return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
	}

	return nil
}

func (e *Engine) decommission(ctx context.Context, r *Reservation) error {
	fn, ok := e.decomissioners[r.Type]
	if !ok {
		return fmt.Errorf("type of reservation not supported: %s", r.Type)
	}

	exists, err := e.cache.Exists(r.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to check if reservation %s exists in cache", r.ID)
	}

	if !exists {
		log.Info().Str("id", r.ID).Msg("reservation not provisioned, no need to decomission")
		if err := e.feedback.Deleted(e.nodeID, r.ID); err != nil {
			log.Error().Err(err).Str("id", r.ID).Msg("failed to mark reservation as deleted")
		}
		return nil
	}

	err = fn(ctx, r)
	if err != nil {
		return errors.Wrap(err, "decommissioning of reservation failed")
	}

	if err := e.cache.Remove(r.ID); err != nil {
		return errors.Wrapf(err, "failed to remove reservation %s from cache", r.ID)
	}

	if err := e.feedback.Deleted(e.nodeID, r.ID); err != nil {
		return errors.Wrap(err, "failed to mark reservation as deleted")
	}

	return nil
}

func (e *Engine) reply(ctx context.Context, r *Reservation, err error, info interface{}) error {
	log.Debug().Str("id", r.ID).Msg("sending reply for reservation")

	result := &Result{
		Type:    r.Type,
		Created: time.Now(),
		ID:      r.ID,
	}
	if err != nil {
		result.Error = err.Error()
		result.State = StateError
	} else {
		result.State = StateOk
	}

	br, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	b, err := result.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed to convert the result to byte for signature")
	}

	sig, err := e.signer.Sign(b)
	if err != nil {
		return errors.Wrap(err, "failed to signed the result")
	}
	result.Signature = hex.EncodeToString(sig)

	return e.feedback.Feedback(e.nodeID, result)
}

func (e *Engine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	ch := make(chan pkg.ProvisionCounters)
	go func() {
		for {
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
			}

			c := e.cache.Counters()
			pc := pkg.ProvisionCounters{
				Container: int64(c.containers.Current()),
				Network:   int64(c.networks.Current()),
				ZDB:       int64(c.zdbs.Current()),
				Volume:    int64(c.volumes.Current()),
				VM:        int64(c.vms.Current()),
			}

			select {
			case <-ctx.Done():
			case ch <- pc:
			}
		}
	}()

	return ch
}
