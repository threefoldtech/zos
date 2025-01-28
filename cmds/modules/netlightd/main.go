package netlightd

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/oasisprotocol/curve25519-voi/primitives/x25519"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zosbase/pkg/netlight"
	"github.com/threefoldtech/zosbase/pkg/netlight/bridge"
	"github.com/threefoldtech/zosbase/pkg/netlight/ifaceutil"
	"github.com/threefoldtech/zosbase/pkg/netlight/nft"
	"github.com/threefoldtech/zosbase/pkg/netlight/public"
	"github.com/threefoldtech/zosbase/pkg/netlight/resource"
	"github.com/urfave/cli/v2"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg/netlight/bootstrap"
	"github.com/threefoldtech/zosbase/pkg/stubs"
	"github.com/threefoldtech/zosbase/pkg/utils"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	module      = "netlight"
)

//go:embed nft/rules.nft
var nftRules embed.FS

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "netlightd",
	Usage: "handles network resources and user networks",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/networkd",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.UintFlag{
			Name:  "workers",
			Usage: "number of workers `N`",
			Value: 1,
		},
	},
	Action: action,
}

func myceliumSeedFromIdentity(privKey []byte) []byte {
	seed := x25519.PrivateKey(x25519.EdPrivateKeyToX25519([]byte(privKey)))
	return seed[:]
}

func action(cli *cli.Context) error {
	var (
		root     string = cli.String("root")
		broker   string = cli.String("broker")
		workerNr uint   = cli.Uint("workers")
	)

	if err := os.MkdirAll(root, 0755); err != nil {
		return errors.Wrap(err, "fail to create module root")
	}

	public.SetPersistence(root)

	waitMyceliumBin()

	if err := bootstrap.DefaultBridgeValid(); err != nil {
		return errors.Wrap(err, "invalid setup")
	}

	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus broker")
	}

	server, err := zbus.NewRedisServer(module, broker, workerNr)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus broker")
	}

	identity := stubs.NewIdentityManagerStub(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	rules, err := nftRules.Open("nft/rules.nft")
	if err != nil {
		return fmt.Errorf("failed to load rules.nft file")
	}

	if err := nft.Apply(rules, ""); err != nil {
		return fmt.Errorf("failed to apply host nft rules: %w", err)
	}
	rules.Close()
	_, err = netlight.CreateNDMZBridge()
	if err != nil {
		return fmt.Errorf("failed to create ndmz bridge: %w", err)
	}

	// create mycelium for host
	hostMyCelium := "hmycelium"
	if !bridge.Exists(resource.HostMyceliumBr) {
		if _, err := bridge.New(resource.HostMyceliumBr); err != nil {
			return fmt.Errorf("could not create bridge %s: %w", resource.HostMyceliumBr, err)
		}
	}
	if !ifaceutil.Exists(hostMyCelium, nil) {
		_, err := ifaceutil.MakeVethPair(hostMyCelium, resource.HostMyceliumBr, 1500, "")
		if err != nil {
			return fmt.Errorf("failed to create mycelium link: %w", err)
		}
	}
	err = resource.SetupMycelium(nil, hostMyCelium, myceliumSeedFromIdentity(identity.PrivateKey(cli.Context)))
	if err != nil {
		return fmt.Errorf("failed to setup mycelium on host: %w", err)
	}

	if err := nft.DropTrafficToLAN(); err != nil {
		return fmt.Errorf("failed to drop traffic to lan: %w", err)
	}

	mod, err := netlight.NewNetworker()
	if err != nil {
		return fmt.Errorf("failed to create Networker: %w", err)
	}
	if err := server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, mod); err != nil {
		return fmt.Errorf("failed to register network light module: %w", err)
	}

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}
	return nil
}

func waitMyceliumBin() {
	log.Info().Msg("wait for mycelium binary to be available")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // forever
	_ = backoff.RetryNotify(func() error {
		_, err := exec.LookPath("mycelium")
		return err
	}, bo, func(err error, d time.Duration) {
		log.Warn().Err(err).Msgf("mycelium binary not found, retying in %s", d.String())
	})
}
