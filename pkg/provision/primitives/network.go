package primitives

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// networkProvision is entry point to provision a network
func (p *Primitives) networkProvisionImpl(ctx context.Context, wl *gridtypes.Workload) error {
	var nr gridtypes.Network
	if err := json.Unmarshal(wl.Data, &nr); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	if err := nr.Valid(); err != nil {
		return fmt.Errorf("validation of the network resource failed: %w", err)
	}

	mgr := stubs.NewNetworkerStub(p.zbus)
	log.Debug().Str("network", fmt.Sprintf("%+v", nr)).Msg("provision network")

	_, err := mgr.CreateNR(nr)
	if err != nil {
		return errors.Wrapf(err, "failed to create network resource for network %s", wl.ID)
	}

	return nil
}

func (p *Primitives) networkProvision(ctx context.Context, wl *gridtypes.Workload) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, wl)
}

func (p *Primitives) networkDecommission(ctx context.Context, wl *gridtypes.Workload) error {
	mgr := stubs.NewNetworkerStub(p.zbus)

	var network gridtypes.Network
	if err := json.Unmarshal(wl.Data, network); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	if err := mgr.DeleteNR(network); err != nil {
		return fmt.Errorf("failed to delete network resource: %w", err)
	}
	return nil
}
