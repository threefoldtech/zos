package zosapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func (g *ZosAPI) networkInterfacesHandler(ctx context.Context, payload []byte) (interface{}, error) {
	results := make(map[string][]net.IPNet)
	type q struct {
		inf    string
		ns     string
		rename string
	}
	interfaces, err := g.networkerLightStub.Interfaces(ctx, "zos", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get ips for 'zos' interface: %w", err)
	}
	zosIfc := interfaces.Interfaces["zos"]
	results["zos"] = zosIfc.IPs

	return results, nil
}

func (g *ZosAPI) networkHasIPv6Handler(ctx context.Context, payload []byte) (interface{}, error) {
	// networkd light
	return false, nil
}

func (g *ZosAPI) networkListPrivateIPsHandler(ctx context.Context, payload []byte) (interface{}, error) {
	var args struct {
		NetworkName gridtypes.Name `json:"network_name"`
	}
	if err := json.Unmarshal(payload, &args); err != nil {
		return nil, err
	}
	twin := peer.GetTwinID(ctx)
	return g.provisionStub.ListPrivateIPs(ctx, twin, args.NetworkName)
}
