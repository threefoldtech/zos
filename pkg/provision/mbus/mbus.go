package mbus

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/mbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	cmdDeploy = "zos.deployment.deploy"
	cmdDelete = "zos.deployment.delete"
)

// WorkloadsMessagebus is provision engine Workloads
type WorkloadsMessagebus struct {
	engine provision.Engine
	mbus   *mbus.MessageBus
}

// NewWorkloadsMessagebus creates a new messagebus instance
func NewWorkloadsMessagebus(engine provision.Engine, address string) (*WorkloadsMessagebus, error) {
	messageBus, err := mbus.New(context.Background(), address)
	if err != nil {
		return nil, err
	}

	api := &WorkloadsMessagebus{
		engine: engine,
		mbus:   messageBus,
	}

	return api, nil
}

func (w *WorkloadsMessagebus) deployHandler(message mbus.Message) error {
	log.Info().Msgf("found deploy request, %s", message.Command)
	_, _ = w.CreateOrUpdate(context.Background(), message, true)
	return nil
}

func (w *WorkloadsMessagebus) deleteHandler(message mbus.Message) error {
	return nil
}

func (w *WorkloadsMessagebus) Run() error {
	w.mbus.Handle(cmdDeploy, w.deployHandler)
	w.mbus.Handle(cmdDelete, w.deleteHandler)

	return nil
}
