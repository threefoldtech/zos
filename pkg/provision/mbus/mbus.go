package mbus

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/rmb"
)

const (
	cmdDeploy = "zos.deployment.deploy"
	cmdDelete = "zos.deployment.delete"
)

// WorkloadsMessagebus is provision engine Workloads
type WorkloadsMessagebus struct {
	engine provision.Engine
	rmb    *rmb.MessageBus
}

// NewWorkloadsMessagebus creates a new messagebus instance
func NewWorkloadsMessagebus(engine provision.Engine, address string) (*WorkloadsMessagebus, error) {
	messageBus, err := rmb.New(context.Background(), address)
	if err != nil {
		return nil, err
	}

	api := &WorkloadsMessagebus{
		engine: engine,
		rmb:    messageBus,
	}

	return api, nil
}

func (w *WorkloadsMessagebus) deployHandler(payload []byte, twinSrc []int) ([]byte, error) {
	log.Info().Msgf("found deploy request, %v", payload)
	_, _ = w.CreateOrUpdate(context.Background(), payload, twinSrc, true)
	return nil, nil
}

func (w *WorkloadsMessagebus) deleteHandler(payload []byte, twinSrc []int) ([]byte, error) {
	return nil, nil
}

// Run runs the messagebus for workloads
func (w *WorkloadsMessagebus) Run() error {
	w.rmb.WithHandler(cmdDeploy, w.deployHandler)
	w.rmb.WithHandler(cmdDelete, w.deleteHandler)

	return w.rmb.Run()
}
