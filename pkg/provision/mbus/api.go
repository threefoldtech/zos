package mbus

import (
	"context"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/rmb"
)

type Deployments struct {
	engine provision.Engine
}

func NewDeploymentMessageBus(router rmb.Router, engine provision.Engine) *Deployments {

	d := Deployments{
		engine: engine,
	}

	d.setup(router)
	return &d
}

func (d *Deployments) setup(router rmb.Router) {
	sub := router.Subroute("deployment")

	// zos deployment handlers
	sub.WithHandler("deploy", d.deployHandler)
	sub.WithHandler("delete", d.deleteHandler)
	sub.WithHandler("get", d.getHandler)
}

func (w *Deployments) deployHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.createOrUpdate(ctx, payload, true)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *Deployments) deleteHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.delete(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *Deployments) getHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.get(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}
