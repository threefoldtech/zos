package zosapi

import (
	"context"
	"encoding/json"
	"fmt"

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
