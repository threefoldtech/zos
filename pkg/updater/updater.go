package updater

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zbus"

	"github.com/threefoldtech/zos/pkg/geoip"
	"github.com/threefoldtech/zos/pkg/registrar"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	updateInterval = 24 * time.Hour
)

type Updater struct {
	substrateGateway *stubs.SubstrateGatewayStub
	registrar        *stubs.RegistrarStub
}

func NewUpdater(bus zbus.Client) *Updater {
	return &Updater{
		substrateGateway: stubs.NewSubstrateGatewayStub(bus),
		registrar:        stubs.NewRegistrarStub(bus),
	}
}

func (u *Updater) Start(ctx context.Context) {
	for {
		if u.registrar.GetState(ctx).State == registrar.Done {
			if err := u.updateLocation(); err != nil {
				log.Error().Err(err).Msg("updating location failed")
			}
			log.Info().Msg("node location updated")
		}

		select {
		case <-ctx.Done():
			log.Info().Msg("stop node updater. context cancelled")
			return
		case <-time.After(updateInterval):
			continue
		}
	}
}

func (u *Updater) updateLocation() error {
	nodeId, err := u.registrar.NodeID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get node id: %w", err)
	}

	node, err := u.substrateGateway.GetNode(context.Background(), nodeId)
	if err != nil {
		return fmt.Errorf("failed to get node from chain: %w", err)
	}

	loc, err := geoip.Fetch()
	if err != nil {
		return fmt.Errorf("failed to fetch location info: %w", err)
	}

	newLoc := substrate.Location{
		City:      loc.City,
		Country:   loc.Country,
		Latitude:  fmt.Sprintf("%f", loc.Latitude),
		Longitude: fmt.Sprintf("%f", loc.Longitude),
	}

	if !reflect.DeepEqual(newLoc, node.Location) {
		node.Location = newLoc
		if _, err := u.substrateGateway.UpdateNode(context.Background(), node); err != nil {
			return fmt.Errorf("failed to update node on chain: %w", err)
		}
	}

	return nil
}
