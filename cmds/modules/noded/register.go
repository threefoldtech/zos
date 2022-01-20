package noded

import (
	"context"
	"crypto/ed25519"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/host"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	reportUptimeEvery = 2 * time.Hour
)

func uptime(ctx context.Context, cl zbus.Client) error {
	var (
		mgr = stubs.NewIdentityManagerStub(cl)
	)
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get runtime environment for zos")
	}

	sub, err := env.GetSubstrate()
	if err != nil {
		return errors.Wrap(err, "failed to create substrate client")
	}

	sk := ed25519.PrivateKey(mgr.PrivateKey(ctx))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return err
	}

	for {
		uptime, err := host.Uptime()
		if err != nil {
			return errors.Wrap(err, "failed to get uptime")
		}
		log.Debug().Msg("updating node uptime")
		hash, err := sub.UpdateNodeUptime(id, uptime)
		if err != nil {
			return errors.Wrap(err, "failed to report uptime")
		}

		log.Info().Str("hash", hash.Hex()).Msg("node uptime hash")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(reportUptimeEvery):
			continue
		}
	}
}
