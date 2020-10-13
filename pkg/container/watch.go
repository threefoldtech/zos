package container

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/events"
	"github.com/containerd/typeurl"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (c *Module) handlerEventTaskExit(ns string, event *events.TaskExit) {

	log := log.With().
		Str("namespace", ns).
		Str("container", event.ContainerID).Logger()

	log.Debug().Msg("task exited")

	if _, ok := c.failures.Get(event.ContainerID); !ok {
		c.failures.Set(event.ContainerID, int(0), cache.DefaultExpiration)
	}

	count, err := c.failures.IncrementInt(event.ContainerID, 1)
	if err != nil {
		// this should never happen because we make sure value
		// is set
		panic(err)
	}

	log.Debug().Int("count", count).Msg("recorded stops")

	if count < failuresBeforeDestroy {
		return
	}

	log.Debug().Msg("deleting container due to so many crashes")

	stub := stubs.NewProvisionStub(c.client)
	if err := stub.DecommissionCached(event.ContainerID, "deleting container due to so many crashes"); err != nil {
		log.Error().Err(err).Msg("failed to decommission reservation")
	}
}

func (c *Module) handleEvent(ns string, event interface{}) {
	switch event := event.(type) {
	case *events.TaskExit:
		c.handlerEventTaskExit(ns, event)
	default:
		log.Debug().Msgf("unhandled event: %+v", event)
	}
}

// watch method will start a connection, and register for events
// once an event is received, it will be handled. and exit on
// first error or in case context was cancelled.
// the caller must make sure this is called again in case of an
// error
func (c *Module) watch(ctx context.Context) error {
	client, err := containerd.New(c.containerd)
	if err != nil {
		return err
	}

	defer client.Close()
	log.Debug().Str("containerd", c.containerd).Msg("subscribe to events from containerd")

	source, errors := client.Subscribe(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case envelope := <-source:
			event, err := typeurl.UnmarshalAny(envelope.Event)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal event envelope")
				continue
			}

			c.handleEvent(envelope.Namespace, event)
		case err := <-errors:
			return err
		}
	}
}

// Watch start watching for events coming from containerd.
// Blocks forever. caller need to run this in a go routine
//
// different events types are handled differently. Now, only
// TaskExit event is handled.
func (c *Module) Watch(ctx context.Context) {
	for {
		err := c.watch(ctx)
		if err == nil {
			break // end of context
		}

		log.Err(err).Msg("error while watching events from containerd")
	}
}
