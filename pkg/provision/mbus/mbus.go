package mbus

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/mbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	cmdDeploy = "msgbus.deploy"
	cmdDelete = "msgbus.delete"
)

// WorkloadsMessagebus is provision engine Workloads
type WorkloadsMessagebus struct {
	engine provision.Engine
	mbus   *mbus.Messagebus
}

// NewWorkloadsMessagebus creates a new messagebus instance
func NewWorkloadsMessagebus(engine provision.Engine, port uint16) (*WorkloadsMessagebus, error) {
	messageBus, err := mbus.New(port, context.Background())
	if err != nil {
		return nil, err
	}

	api := &WorkloadsMessagebus{
		engine: engine,
		mbus:   messageBus,
	}

	return api, nil
}

func (w *WorkloadsMessagebus) Run() error {
	deployChan := make(chan mbus.Message)
	deleteChan := make(chan mbus.Message)

	go func() {
		if err := w.mbus.StreamMessages(context.Background(), cmdDeploy, deployChan); err != nil {
			panic(err)
		}
		if err := w.mbus.StreamMessages(context.Background(), cmdDelete, deleteChan); err != nil {
			panic(err)
		}
	}()

	select {
	case deploy := <-deployChan:
		log.Info().Msgf("found deploy request, %s", deploy.Command)
		_, _ = w.CreateOrUpdate(context.Background(), deploy, true)
		// TODO handle reply

	case delete := <-deleteChan:
		log.Info().Msgf("found delete request, %s", delete.Command)
	}

	return nil
}
