package container

import (
	"context"

	"fmt"
	"net/url"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/namespaces"
)

const (
	restartFlag = "threefoldtech/restart.enabled"
)

func killTask(ctx context.Context, container containerd.Container) error {
	task, err := container.Task(ctx, nil)
	if err == nil {
		wait, err := task.Wait(ctx)
		if err != nil {
			if _, derr := task.Delete(ctx); derr == nil {
				return nil
			}
			return err
		}
		if err := task.Kill(ctx, syscall.SIGKILL, containerd.WithKillAll); err != nil {
			if _, derr := task.Delete(ctx); derr == nil {
				return nil
			}
			return err
		}
		<-wait
		if _, err := task.Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}

func containerStatus(ctx context.Context, container containerd.Container) error {
	task, err := container.Task(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch container tasks")
		return err
	}

	state, err := task.Status(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch task status")
		return err
	}

	log.Debug().Str("id", container.ID()).Str("status", string(state.Status)).Msg("container status")

	if state.Status == containerd.Stopped {
		err := containerStopped(ctx, container)

		if err != nil {
			log.Error().Err(err).Str("id", container.ID()).Msg("could not restart container")
			return err
		}
	}

	return nil
}

func containerStopped(ctx context.Context, container containerd.Container) error {
	log.Info().Str("id", container.ID()).Msg("container not running, restarting it")

	killTask(ctx, container)

	// setting external logger process
	uri, err := url.Parse("binary:///bin/shim-logs")
	if err != nil {
		log.Error().Err(err).Msg("could not parse shim-logs uri")
		return err
	}

	// task, err := c.NewTask(nsctx, cio.NullIO)
	task, err := container.NewTask(ctx, cio.LogURI(uri))
	if err != nil {
		log.Error().Err(err).Msg("could not create new task")
		return err
	}

	if err := task.Start(ctx); err != nil {
		log.Error().Err(err).Msg("could not start new task")
		return err
	}

	return nil
}

// Maintenance work in the background and do actions at specific interval
func (c *containerModule) Maintenance() error {
	client, err := containerd.New(c.containerd)
	if err != nil {
		log.Error().Err(err).Msg("could not connect containerd socket")
		return err
	}
	defer client.Close()

	ctx := context.Background()

	for {
		log.Debug().Msg("container scrub loop")

		ns, err := client.NamespaceService().List(ctx)
		if err != nil {
			log.Error().Err(err).Msg("could not fetch containerd namespaces list")
			continue
		}

		for _, nsname := range ns {
			nsctx := namespaces.WithNamespace(context.Background(), nsname)

			containers, err := client.Containers(nsctx, fmt.Sprintf("labels.%q", restartFlag))
			if err != nil {
				log.Error().Err(err).Msg("containers restart label")
				continue
			}

			for _, c := range containers {
				containerStatus(nsctx, c)
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func ensureLabels(c *containers.Container) {
	if c.Labels == nil {
		c.Labels = make(map[string]string)
	}
}

func withRestart() func(context.Context, *containerd.Client, *containers.Container) error {
	return func(_ context.Context, _ *containerd.Client, c *containers.Container) error {
		ensureLabels(c)
		c.Labels[restartFlag] = "yes"
		return nil
	}
}
