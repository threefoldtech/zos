package container

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	zbusClient zbus.Client
)

// InitUptimeReporter inits the uptime reporter
func InitUptimeReporter(client zbus.Client) {
	zbusClient = client
}

// SendUptime sends uptime of the node to bcdb
func SendUptime(ctx context.Context, id pkg.Identifier, directoryClient client.Directory) error {
	storage := stubs.NewStorageModuleStub(zbusClient)

	r := capacity.NewResourceOracle(storage)

	sendUptime := func() error {
		uptime, err := r.Uptime()
		if err != nil {
			log.Error().Err(err).Msgf("failed to read uptime")
			return err
		}

		log.Info().Msg("send heart-beat to BCDB")
		if err := directoryClient.NodeUpdateUptime(id.Identity(), uptime); err != nil {
			log.Error().Err(err).Msgf("failed to send heart-beat to BCDB")
			return err
		}
		return nil
	}
	if err := sendUptime(); err != nil {
		log.Fatal().Err(err).Send()
	}

	tick := time.NewTicker(time.Minute * 10)

	go func() {
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				backoff.Retry(sendUptime, backoff.NewExponentialBackOff())
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}
