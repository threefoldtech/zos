package mbus

import (
	"context"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/primitives"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/rmb"
)

const (
	cmdDeploy          = "zos.deployment.deploy"
	cmdDelete          = "zos.deployment.delete"
	cmdGet             = "zos.deployment.get"
	cmdListPorts       = "zos.network.list_wg_ports"
	cmdListIPs         = "zos.network.list_public_ips"
	cmdGetPublicConfig = "zos.network.public_config_get"
	cmdSetPublicConfig = "zos.network.public_config_set"
	cmdGetStatistics   = "zos.statistics.get"
)

// WorkloadsMessagebus is provision engine Workloads
type WorkloadsMessagebus struct {
	engine provision.Engine
	rmb    *rmb.MessageBus
	cl     zbus.Client
	stats  *primitives.Statistics
}

// NewWorkloadsMessagebus creates a new messagebus instance
func NewWorkloadsMessagebus(engine provision.Engine, cl zbus.Client, stats primitives.Statistics, address string) (*WorkloadsMessagebus, error) {
	messageBus, err := rmb.New(context.Background(), address)
	if err != nil {
		return nil, err
	}

	api := &WorkloadsMessagebus{
		engine: engine,
		rmb:    messageBus,
		cl:     cl,
		stats:  &stats,
	}

	return api, nil
}

func (w *WorkloadsMessagebus) deployHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.createOrUpdate(ctx, payload, true)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) deleteHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.delete(ctx, payload)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) getHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.get(ctx, payload)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) listPortsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.listPorts(ctx)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) listPublicIPsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.listPublicIps()
	return data, err.Err()
}

func (w *WorkloadsMessagebus) getPublicConfigHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.getPublicConfig(ctx)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) setPublicConfigHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.setPublicConfig(ctx, payload)
	return data, err.Err()
}

func (w *WorkloadsMessagebus) getStatisticsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	data, err := w.getStatistics(ctx)
	return data, err.Err()
}

// Run runs the messagebus for workloads
func (w *WorkloadsMessagebus) Run() error {
	// zos deployment handlers
	w.rmb.WithHandler(cmdDeploy, w.deployHandler)
	w.rmb.WithHandler(cmdDelete, w.deleteHandler)
	w.rmb.WithHandler(cmdGet, w.getHandler)

	// network handlers
	w.rmb.WithHandler(cmdListPorts, w.listPortsHandler)
	w.rmb.WithHandler(cmdListIPs, w.listPublicIPsHandler)
	w.rmb.WithHandler(cmdGetPublicConfig, w.getPublicConfigHandler)
	w.rmb.WithHandler(cmdSetPublicConfig, w.setPublicConfigHandler)

	// statistics handlers
	w.rmb.WithHandler(cmdGetStatistics, w.getStatisticsHandler)

	return w.rmb.Run(context.Background())
}
