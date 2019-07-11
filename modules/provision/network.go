package provision

import (
	"context"
	"encoding/json"

	"github.com/threefoldtech/zosv2/modules/stubs"

	"github.com/threefoldtech/zosv2/modules"
)

// VolumeProvision is entry point to provision a volume
func NetworkProvision(ctx context.Context, reservation Reservation) (interface{}, error) {
	var netID modules.NetID
	if err := json.Unmarshal(reservation.Data, &netID); err != nil {
		return nil, err
	}

	db := GetTnoDB(ctx)
	no, err := db.GetNetwork(netID)
	if err != nil {
		return nil, err
	}

	mgr := stubs.NewNetworkerStub(GetZBus(ctx))
	if err := mgr.ApplyNetResource(*no); err != nil {
		return nil, err
	}

	return nil, nil
}
