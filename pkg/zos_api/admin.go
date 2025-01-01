package zosapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/threefoldtech/zos/pkg/zinit"
)

func (g *ZosAPI) adminRestartServiceHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var service string
	if err := json.Unmarshal(payload, &service); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting string: %w", err)
	}

	zinit := zinit.Default()

	return nil, zinit.Restart(service)
}

func (g *ZosAPI) adminRebootHandler(ctx context.Context, payload []byte) (interface{}, error) {
	zinit := zinit.Default()

	return nil, zinit.Reboot()
}

func (g *ZosAPI) adminRestartAllHandler(ctx context.Context, payload []byte) (interface{}, error) {
	zinit := zinit.Default()

	services, err := zinit.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list node services, expecting string: %w", err)
	}

	for _, service := range services {
		if err := zinit.Restart(service.String()); err != nil {
			return nil, fmt.Errorf("failed to reboot service %s, expecting string: %w", service.String(), err)
		}
	}

	return nil, nil
}

func (g *ZosAPI) adminShowLogsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var n int
	if err := json.Unmarshal(payload, &n); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting string: %w", err)
	}

	zinit := zinit.Default()

	return zinit.Log(n)
}

func (g *ZosAPI) adminShowResolveHandler(ctx context.Context, payload []byte) (interface{}, error) {
	path := filepath.Join("/etc", "resolv.conf")
	return os.ReadFile(path)
}

func (g *ZosAPI) adminShowOpenConnectionsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.statisticsStub.OpenConnections(ctx)
}

func (g *ZosAPI) adminStopWorkloadHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID     uint32 `json:"twin_id"`
		WorkloadID uint64 `json:"workload_id"`
	}

	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting twin id and workload id: %w", err)
	}

	return nil, g.provisionStub.Pause(ctx, args.TwinID, args.WorkloadID)
}

func (g *ZosAPI) adminResumeWorkloadHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		TwinID     uint32 `json:"twin_id"`
		WorkloadID uint64 `json:"workload_id"`
	}

	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting twin id and workload id: %w", err)
	}

	return nil, g.provisionStub.Resume(ctx, args.TwinID, args.WorkloadID)
}

func (g *ZosAPI) adminInterfacesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	// list all interfaces on node
	type Interface struct {
		IPs []string `json:"ips"`
		Mac string   `json:"mac"`
	}

	interfaces, err := g.networkerStub.Interfaces(ctx, "", "")
	if err != nil {
		return nil, err
	}
	output := make(map[string]Interface)
	for name, inf := range interfaces.Interfaces {
		output[name] = Interface{
			Mac: inf.Mac,
			IPs: func() []string {
				var ips []string
				for _, ip := range inf.IPs {
					ips = append(ips, ip.String())
				}
				return ips
			}(),
		}
	}

	return output, nil
}

func (g *ZosAPI) adminGetPublicNICHandler(ctx context.Context, payload []byte) (interface{}, error) {
	return g.networkerStub.GetPublicExitDevice(ctx)
}

func (g *ZosAPI) adminSetPublicNICHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var iface string
	if err := json.Unmarshal(payload, &iface); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting string: %w", err)
	}
	return nil, g.networkerStub.SetPublicExitDevice(ctx, iface)
}
