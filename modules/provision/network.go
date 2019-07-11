package provision

import (
	"context"
	"encoding/json"

	"github.com/threefoldtech/zosv2/modules/stubs"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/network/ip"
)

func networkProvision(ctx context.Context, netID modules.NetID) (network *modules.Network, err error) {
	db := GetTnoDB(ctx)
	network, err = db.GetNetwork(netID)
	if err != nil {
		return nil, err
	}

	mgr := stubs.NewNetworkerStub(GetZBus(ctx))
	_, err = mgr.GenerateWireguarKeyPair(network.NetID)

	if err != nil {
		return nil, err
	}

	if err := mgr.ApplyNetResource(*network); err != nil {
		return nil, err
	}

	return nil, err
}

func networkGetNamespace(network *modules.Network) string {
	nib := ip.NewNibble(network.PrefixZero, network.AllocationNR)
	return nib.NetworkName()
}

// NetworkProvision is entry point to provision a network
func NetworkProvision(ctx context.Context, reservation Reservation) (interface{}, error) {
	var netID modules.NetID
	if err := json.Unmarshal(reservation.Data, &netID); err != nil {
		return nil, err
	}

	_, err := networkProvision(ctx, netID)
	return nil, err
}
