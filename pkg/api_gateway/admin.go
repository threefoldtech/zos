package apigateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

func (g *apiGateway) adminInterfacesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	user := peer.GetTwinID(ctx)
	if user != g.farmerID {
		return nil, fmt.Errorf("unauthorized")
	}
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

func (g *apiGateway) adminGetPublicNICHandler(ctx context.Context, payload []byte) (interface{}, error) {
	user := peer.GetTwinID(ctx)
	if user != g.farmerID {
		return nil, fmt.Errorf("unauthorized")
	}
	return g.networkerStub.GetPublicExitDevice(ctx)
}

func (g *apiGateway) adminSetPublicNICHandler(ctx context.Context, payload []byte) (interface{}, error) {
	user := peer.GetTwinID(ctx)
	if user != g.farmerID {
		return nil, fmt.Errorf("unauthorized")
	}
	var iface string
	if err := json.Unmarshal(payload, &iface); err != nil {
		return nil, fmt.Errorf("failed to decode input, expecting string: %w", err)
	}
	return nil, g.networkerStub.SetPublicExitDevice(ctx, iface)
}
