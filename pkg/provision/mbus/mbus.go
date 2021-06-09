package mbus

import (
	"context"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/rmb"
)

const (
	cmdDeploy = "zos.deployment.deploy"
	cmdDelete = "zos.deployment.delete"
	cmdGet    = "zos.deployment.get"
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

func (w *WorkloadsMessagebus) deployHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.CreateOrUpdate(ctx, payload, true)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) deleteHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.Delete(ctx, payload)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) getHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.Get(ctx, payload)
	return data, err.Err()
}

// Run runs the messagebus for workloads
func (w *WorkloadsMessagebus) Run() error {
	w.rmb.WithHandler(cmdDeploy, w.deployHandler)
	w.rmb.WithHandler(cmdDelete, w.deleteHandler)
	w.rmb.WithHandler(cmdGet, w.getHandler)

	return w.rmb.Run(context.Background())
}
