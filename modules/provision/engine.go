package provision

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

type ReservationReadWriter interface {
	Add(r *Reservation) error
	Remove(id string) error
}

type defaultEngine struct {
	source ReservationSource
	store  ReservationReadWriter
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(source ReservationSource, rw ReservationReadWriter) Engine {
	return &defaultEngine{
		source: source,
		store:  rw,
	}
}

// Run starts processing reservation resource. Then try to allocate
// reservations
func (e *defaultEngine) Run(ctx context.Context) error {

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

			slog := log.With().
				Str("id", string(reservation.ID)).
				Str("type", string(reservation.Type)).
				Str("duration", fmt.Sprintf("%v", reservation.Duration)).
				Logger()

			if !reservation.expired() {

				if err := reservation.validate(); err != nil {
					log.Error().Err(err).Msgf("failed validation of reservation")
					continue
				}

				// provisioning of a reservation
				slog.Info().Msg("start provisioning reservation")

				fn, ok := provisioners[reservation.Type]
				if !ok {
					slog.Error().Msg("reservation: type of reservation not supported")
					continue
				}

				result, err := fn(ctx, reservation)
				if err != nil {
					slog.Error().Err(err).Msg("provisioning of reservation failed")
				}

				if err := e.store.Add(reservation); err != nil {
					log.Error().Err(err).Msgf("failed to cache reservation %s locally", reservation.ID)
				}

				e.reply(reservation.ID, result, err)
			} else {
				// here we handle the case when the reservation is expired
				slog.Info().Msg("start decommissioning reservation")

				fn, ok := decommissioners[reservation.Type]
				if !ok {
					slog.Error().Msg("decommission: type of reservation not supported")
					continue
				}

				err := fn(ctx, reservation)
				if err != nil {
					slog.Error().Err(err).Msg("decommissioning of reservation failed")
				}

				if err := e.store.Remove(reservation.ID); err != nil {
					log.Error().Err(err).Msgf("failed to remove reservation %s from cache", reservation.ID)
				}
			}
		}
	}
}

func (e *defaultEngine) reply(id string, result interface{}, err error) {
	//TODO: actually push the reply to the endpoint defined by `to`
	if err != nil {
		log.Error().
			Err(err).
			Str("id", id).
			Msgf("failed to apply provision")

		return
	}

	log.Info().Str("reservation", id).Str("result", fmt.Sprint(result)).Msg("reservation result")
}
