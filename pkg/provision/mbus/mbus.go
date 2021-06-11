package mbus

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/primitives"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// WorkloadsMessagebus is provision engine Workloads
type WorkloadsMessagebus struct {
	engine provision.Engine
	rmb    *rmb.MessageBus
	cl     zbus.Client
	stats  *primitives.Statistics
}

// NewWorkloadsMessagebus creates a new messagebus instance
func NewWorkloadsMessagebus(engine provision.Engine, cl zbus.Client, stats *primitives.Statistics, address string) (*WorkloadsMessagebus, error) {
	messageBus, err := rmb.New(context.Background(), address)
	if err != nil {
		return nil, err
	}

	api := &WorkloadsMessagebus{
		engine: engine,
		rmb:    messageBus,
		cl:     cl,
		stats:  stats,
	}

	return api, nil
}

func (w *WorkloadsMessagebus) deployHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.createOrUpdate(ctx, payload, true)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *WorkloadsMessagebus) deleteHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.delete(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *WorkloadsMessagebus) getHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.get(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *WorkloadsMessagebus) listPortsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.listPorts(ctx)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *WorkloadsMessagebus) listPublicIPsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.listPublicIps()
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *WorkloadsMessagebus) getPublicConfigHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.getPublicConfig(ctx)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *WorkloadsMessagebus) setPublicConfigHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.setPublicConfig(ctx, payload)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

func (w *WorkloadsMessagebus) getStatisticsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.getStatistics(ctx)
	if err != nil {
		return nil, err.Err()
	}
	return data, nil
}

// Run runs the messagebus for workloads
func (w *WorkloadsMessagebus) Run() error {
	msgBusRouter := w.rmb.Subroute("zos")

	// zos deployment handlers
	zosRouter := msgBusRouter.Subroute("deployment")
	zosRouter.WithHandler("deploy", w.deployHandler)
	zosRouter.WithHandler("delete", w.deleteHandler)
	zosRouter.WithHandler("get", w.getHandler)

	// network handlers
	networkRouter := msgBusRouter.Subroute("network")
	networkRouter.WithHandler("list_wg_ports", w.listPortsHandler)
	networkRouter.WithHandler("list_public_ips", w.listPublicIPsHandler)
	networkRouter.WithHandler("public_config_get", w.getPublicConfigHandler)
	networkRouter.WithHandler("public_config_set", w.setPublicConfigHandler)

	// statistics handlers
	statsRouter := msgBusRouter.Subroute("statistics")
	statsRouter.WithHandler("get", w.getStatisticsHandler)

	log.Debug().Msg("messagebus is running...")
	return w.rmb.Run(context.Background())
}
