package capacityd

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/farmer"
	"github.com/threefoldtech/zos/pkg/geoip"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/substrate"
)

func registration(ctx context.Context, cl zbus.Client) error {
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get runtime environment for zos")
	}

	mgr := stubs.NewIdentityManagerStub(cl)
	storage := stubs.NewStorageModuleStub(cl)

	loc, err := geoip.Fetch()
	if err != nil {
		log.Fatal().Err(err).Msg("fetch location")
	}
	farmIP, err := getFarmTwin(env)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get farmer twin address")
	}

	log.Debug().IPAddr("ip", farmIP).Msg("farmer IP")
	oracle := capacity.NewResourceOracle(storage)
	cap, err := oracle.Total()
	if err != nil {
		return errors.Wrap(err, "failed to get node capacity")
	}

	log.Debug().
		Uint64("cru", cap.CRU).
		Uint64("mru", cap.MRU).
		Uint64("sru", cap.SRU).
		Uint64("hru", cap.HRU).
		Msg("node capacity")

	url := fmt.Sprintf("http://[%s]:3000/", farmIP.String())
	fm, err := farmer.NewClient(url)
	if err != nil {
		return errors.Wrap(err, "failed to create farmer client")
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Minute
	bo := backoff.WithContext(exp, ctx)
	err = backoff.RetryNotify(func() error {
		return registerNode(env, mgr, fm, cap, loc)
	}, bo, retryNotify)

	if err != nil {
		return errors.Wrap(err, "failed to register node")
	}

	log.Info().Msg("node has been registered")
	return nil
}

func retryNotify(err error, d time.Duration) {
	log.Warn().Err(err).Str("sleep", d.String()).Msg("registration failed")
}

func registerNode(env environment.Environment, mgr pkg.IdentityManager, cl *farmer.Client, cap capacity.Capacity, loc geoip.Location) error {
	log.Info().Str("id", mgr.NodeID().Identity()).Msg("start registration of the node")
	log.Info().Msg("registering at farmer bot")

	hostName, err := os.Hostname()
	if err != nil {
		hostName = "unknown"
	}

	return cl.NodeRegister(farmer.Node{
		ID:       mgr.NodeID().Identity(),
		HostName: hostName,
		FarmID:   uint32(env.FarmerID),
		Secret:   env.FarmSecret,
		Location: farmer.Location{
			Continent: loc.Continent,
			Country:   loc.Country,
			City:      loc.City,
			Longitude: loc.Longitute,
			Latitude:  loc.Latitude,
		},
		Capacity: cap,
	})
}

func getFarmTwin(env environment.Environment) (net.IP, error) {
	sub, err := substrate.NewSubstrate(env.SubstrateURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to substrate")
	}

	farm, err := sub.GetFarm(uint32(env.FarmerID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get farm '%d'", env.FarmerID)
	}

	twin, err := sub.GetTwin(uint32(farm.TwinID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get twin")
	}

	ip := twin.IPAddress()
	if len(ip) == 0 {
		return nil, fmt.Errorf("invalid ip address associated with farmer twin")
	}

	return ip, nil
}
