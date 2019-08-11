package provision

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type defaultEngine struct {
	source ReservationSource
	store  LocalStore
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(source ReservationSource) Engine {
	return &defaultEngine{
		source: source,
		store:  NewMemStore(),
	}
}

// Run starts processing reservation resource. Then try to allocate
// reservations
func (e *defaultEngine) Run(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cReservation := e.iterReservation(ctx)
	cExpiration := e.iterExpiration(ctx)

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

			e.store.Add(&reservation)
			e.reply(reservation.ID, result, err)

		case reservation := <-cExpiration:
			slog := log.With().
				Str("id", string(reservation.ID)).
				Str("type", string(reservation.Type)).
				Logger()
			slog.Info().Msg("start decomissioning reservation")

			fn, ok := decomissioners[reservation.Type]
			if !ok {
				slog.Error().Msg("decomission: type of reservation not supported")
				continue
			}

			err := fn(ctx, *reservation)
			if err != nil {
				slog.Error().Err(err).Msg("decomissioning of reservation failed")
			}
			if err := e.store.Remove(reservation.ID); err != nil {
				log.Error().Err(err).Msg("failed to remove reservation %s from expiration store")
			}
		}
	}

}

func (e *defaultEngine) iterReservation(ctx context.Context) <-chan Reservation {
	c := make(chan Reservation)

	go func() {
		defer close(c)

		for reservation := range e.source.Reservations(ctx) {
			log.Info().
				Str("id", string(reservation.ID)).
				Str("type", string(reservation.Type)).
				Msg("reservation received")

			if err := reservation.validate(); err != nil {
				log.Error().Err(err).Msgf("failed validation of reservation")
				continue
			}

			select {
			case <-ctx.Done():
				return
			case c <- reservation:
			}
		}
	}()

	return c
}

func (e *defaultEngine) iterExpiration(ctx context.Context) <-chan *Reservation {
	c := make(chan *Reservation)

	go func() {
		defer close(c)

		for {
			<-time.After(time.Minute * 10) //TODO: make configuration ? default value ?
			log.Info().Msg("check for expired reservation")

			reservations, err := e.store.GetExpired()
			if err != nil {
				log.Error().Err(err).Msg("error while getting expired reservation id")
				continue
			}

			select {
			case <-ctx.Done():
				return
			default:
				for _, r := range reservations {
					log.Info().
						Str("id", string(r.ID)).
						Str("type", string(r.Type)).
						Time("created", r.Created).
						Str("duration", fmt.Sprintf("%v", r.Duration)).
						Msg("reservation expired")
					c <- r
				}
			}
		}
	}()
	return c
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
