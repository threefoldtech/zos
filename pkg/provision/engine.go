package provision

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/jbenet/go-base58"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const gib = 1024 * 1024 * 1024

const minimunZosMemory = 2 * gib

// Provisioner interface
type Provisioner interface {
	Provision(ctx context.Context, reservation *Reservation) (*Result, error)
	Decommission(ctx context.Context, reservation *Reservation) error
}

// Engine is the core of this package
// The engine is responsible to manage provision and decomission of workloads on the system
type Engine struct {
	source ReservationSource

	provisioner Provisioner
	janitor     *Janitor

	memCache          *cache.Cache
	totalMemAvailable uint64
}

// EngineOps are the configuration of the engine
type EngineOps struct {
	// Source is responsible to retrieve reservation for a remote source
	Source ReservationSource

	Provisioner Provisioner

	// Janitor is used to clean up some of the resources that might be lingering on the node
	// if not set, no cleaning up will be done
	Janitor *Janitor
}

// New creates a new engine. Once started, the engine
// will continue processing all reservations from the reservation source
// and try to apply them.
// the default implementation is a single threaded worker. so it process
// one reservation at a time. On error, the engine will log the error. and
// continue to next reservation.
func New(opts EngineOps) (*Engine, error) {
	memStats, err := mem.VirtualMemory()
	if err != nil {
		return nil, errors.Wrap(err, "failed retrieve memory stats")
	}

	// we round the total memory size to the nearest 1G
	totalMemory := math.Ceil(float64(memStats.Total)/gib) * gib

	return &Engine{
		source:      opts.Source,
		provisioner: opts.Provisioner,
		janitor:     opts.Janitor,

		memCache:          cache.New(30*time.Minute, 30*time.Second),
		totalMemAvailable: uint64(totalMemory) - minimunZosMemory,
	}, nil
}

// Run starts reader reservation from the Source and handle them
func (e *Engine) Run(ctx context.Context) error {
	cReservation := e.source.Reservations(ctx)

	isAllWorkloadsProcessed := false
	// run a cron task that will fire the cleanup at midnight
	cleanUp := make(chan struct{}, 2)
	c := cron.New()
	_, err := c.AddFunc("@midnight", func() {
		cleanUp <- struct{}{}
	})
	if err != nil {
		return fmt.Errorf("failed to setup cron task: %w", err)
	}

	c.Start()
	defer c.Stop()

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

			if reservation.last {
				isAllWorkloadsProcessed = true
				// Trigger cleanup by sending a struct onto the channel
				log.Debug().Msg("kicking clean up after redeploying history")
				cleanUp <- struct{}{}
				continue
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
				if err := e.decommission(ctx, &reservation.Reservation); err != nil {
					log.Error().Err(err).Msg("failed to decommission reservation")
					continue
				}
			} else {
				slog.Info().Msg("start provisioning reservation")

				//TODO:
				// this is just a hack now to avoid having double provisioning
				// other logs has been added in other places so we can find why
				// the node keep receiving the same reservation twice
				if _, ok := e.memCache.Get(reservation.ID); ok {
					log.Debug().Str("id", reservation.ID).Msg("skipping reservation since it has just been processes!")
					continue
				}

				e.memCache.Set(reservation.ID, struct{}{}, cache.DefaultExpiration)

				if err := e.provision(ctx, &reservation.Reservation); err != nil {
					log.Error().Err(err).Msg("failed to provision reservation")
					continue
				}
			}

		case <-cleanUp:
			if !isAllWorkloadsProcessed {
				// only allow cleanup triggered by the cron to run once
				// we are sure all the workloads from the cache/explorer have been processed
				log.Info().Msg("all workloads not yet processed, delay cleanup")
				continue
			}
			log.Info().Msg("start cleaning up resources")
			if e.janitor == nil {
				log.Info().Msg("janitor is not configured, skipping clean up")
				continue
			}

			if err := e.janitor.CleanupResources(ctx); err != nil {
				log.Error().Err(err).Msg("failed to cleanup resources")
				continue
			}
		}
	}
}

func (e *Engine) provision(ctx context.Context, reservation *Reservation) error {
	if err := reservation.validate(); err != nil {
		return errors.Wrapf(err, "failed validation of reservation")
	}

	if _, err := e.provisioner.Provision(ctx, reservation); err != nil {
		return err
	}

	// we only cache successful reservations
	// r.ID = realID
	// r.Result = *result
	// if err := e.cache.Add(r, false); err != nil {
	// 	return errors.Wrapf(err, "failed to cache reservation %s locally", r.ID)
	// }

	// // If an update occurs on the network we don't increment the counter
	// if r.Type == "network_resource" {
	// 	nr := pkg.NetResource{}
	// 	if err := json.Unmarshal(r.Data, &nr); err != nil {
	// 		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	// 	}

	// 	uniqueID := NetworkID(r.User, nr.Name)
	// 	exists, err := e.cache.NetworkExists(string(uniqueID))
	// 	if err != nil {
	// 		return errors.Wrap(err, "failed to check if network exists")
	// 	}
	// 	if exists {
	// 		return nil
	// 	}
	// }

	return nil
}

func (e *Engine) provisionForward(ctx context.Context, r *Reservation) (interface{}, error) {
	if err := e.statser.CheckMemoryRequirements(r, e.totalMemAvailable); err != nil {
		return nil, errors.Wrapf(err, "failed to apply provision")
	}
	returned, provisionError := e.provisioner.Provision(ctx, r)
	if provisionError != nil {
		log.Error().
			Err(provisionError).
			Str("id", r.ID).
			Msgf("failed to apply provision")
	} else {
		log.Info().
			Str("result", fmt.Sprintf("%v", returned)).
			Msgf("workload deployed")
	}
	return returned, nil
}

func (e *Engine) decommission(ctx context.Context, r *Reservation) error {
	return e.provisioner.Decommission(ctx, r)
}

// DecommissionCached is used by other module to ask provisiond that
// a certain reservation is dead beyond repair and owner must be informed
// the decommission method will take care to update the reservation instance
// and also decommission the reservation normally
func (e *Engine) DecommissionCached(id string, reason string) error {
	return fmt.Errorf("not implemented")
	// r, err := e.cache.Get(id)
	// if err != nil {
	// 	return err
	// }

	// ctx := context.Background()
	// result, err := e.buildResult(id, r.Type, fmt.Errorf(reason), nil)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to build result object for reservation: %s", id)
	// }

	// if err := e.decommission(ctx, r); err != nil {
	// 	log.Error().Err(err).Msgf("failed to update reservation result with failure: %s", id)
	// }

	// bf := backoff.NewExponentialBackOff()
	// bf.MaxInterval = 10 * time.Second
	// bf.MaxElapsedTime = 1 * time.Minute

	// return backoff.Retry(func() error {
	// 	err := e.reply(ctx, result)
	// 	if err != nil {
	// 		log.Error().Err(err).Msgf("failed to update reservation result with failure: %s", id)
	// 	}
	// 	return err
	// }, bf)
}

func (e *Engine) buildResult(id string, typ ReservationType, err error, info interface{}) (*Result, error) {
	result := &Result{
		Type:    typ,
		Created: time.Now(),
		ID:      id,
	}

	if err != nil {
		result.Error = err.Error()
		result.State = StateError
	} else {
		result.State = StateOk
	}

	br, err := json.Marshal(info)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode result")
	}
	result.Data = br

	return result, nil
}

// func (e *Engine) updateStats() error {
// 	wl := e.statser.CurrentWorkloads()
// 	r := e.statser.CurrentUnits()

// 	if e.zbusCl != nil {
// 		// TODO: this is a very specific zos code that should not be
// 		// here. this is a quick fix for the tfgateways
// 		// but should be implemented cleanely after
// 		storaged := stubs.NewStorageModuleStub(e.zbusCl)

// 		cache, err := storaged.GetCacheFS()
// 		if err != nil {
// 			return err
// 		}

// 		switch cache.DiskType {
// 		case pkg.SSDDevice:
// 			r.Sru += float64(cache.Usage.Size / gib)
// 		case pkg.HDDDevice:
// 			r.Hru += float64(cache.Usage.Size / gib)
// 		}

//		r.Mru += float64(minimunZosMemory / gib)
//	}

// 	log.Info().
// 		Uint16("network", wl.Network).
// 		Uint16("volume", wl.Volume).
// 		Uint16("zDBNamespace", wl.ZDBNamespace).
// 		Uint16("container", wl.Container).
// 		Uint16("k8sVM", wl.K8sVM).
// 		Uint16("proxy", wl.Proxy).
// 		Uint16("reverseProxy", wl.ReverseProxy).
// 		Uint16("subdomain", wl.Subdomain).
// 		Uint16("delegateDomain", wl.DelegateDomain).
// 		Uint64("cru", r.Cru).
// 		Float64("mru", r.Mru).
// 		Float64("hru", r.Hru).
// 		Float64("sru", r.Sru).
// 		Msgf("provision statistics")

// 	return e.feedback.UpdateStats(e.nodeID, wl, r)
// }

func (e *Engine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
	return nil
}

// // Counters is a zbus stream that sends statistics from the engine
// func (e *Engine) Counters(ctx context.Context) <-chan pkg.ProvisionCounters {
// 	ch := make(chan pkg.ProvisionCounters)
// 	go func() {
// 		for {
// 			select {
// 			case <-time.After(2 * time.Second):
// 			case <-ctx.Done():
// 			}

// 			wls := e.statser.CurrentWorkloads()
// 			pc := pkg.ProvisionCounters{
// 				Container: int64(wls.Container),
// 				Network:   int64(wls.Network),
// 				ZDB:       int64(wls.ZDBNamespace),
// 				Volume:    int64(wls.Volume),
// 				VM:        int64(wls.K8sVM),
// 			}

// 			select {
// 			case <-ctx.Done():
// 			case ch <- pc:
// 			}
// 		}
// 	}()

// 	return ch
// }

// NetworkID construct a network ID based on a userID and network name
func NetworkID(userID, name string) pkg.NetID {
	buf := bytes.Buffer{}
	buf.WriteString(userID)
	buf.WriteString(name)
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return pkg.NetID(string(b))
}
