package netlightd

import (
	"context"
	"embed"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/oasisprotocol/curve25519-voi/primitives/x25519"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/netlight"
	"github.com/threefoldtech/zos/pkg/netlight/nft"
	"github.com/threefoldtech/zos/pkg/netlight/resource"
	"github.com/urfave/cli/v2"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/netlight/bootstrap"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
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

	bridge, err := netlight.CreateNDMZBridge()
	if err != nil {
		return fmt.Errorf("failed to create ndmz bridge: %w", err)
	}

	_, err = resource.Create("dmz", bridge, &net.IPNet{
		IP:   net.ParseIP("100.127.0.2"),
		Mask: net.CIDRMask(16, 32),
	}, netlight.NDMZGwIP, nil, myceliumSeedFromIdentity(identity.PrivateKey(cli.Context)))

	if err != nil {
		return fmt.Errorf("failed to create ndmz resource: %w", err)
	}

	// create a test user network
	// r, err := resource.Create("test", bridge, &net.IPNet{
	// 	IP:   net.ParseIP("100.127.0.10"),
	// 	Mask: net.CIDRMask(16, 32),
	// }, netlight.NDMZGwIP, &net.IPNet{
	// 	IP:   net.ParseIP("192.168.1.0"),
	// 	Mask: net.CIDRMask(24, 32),
	// }, zos.MustBytesFromHex("8ad7d29b81df3f3ef0a5ff95c25cc0824ef33137fbbcf22d2f23b0222ae3ac00"))

	// if err != nil {
	// 	return fmt.Errorf("failed to create user resource: %w", err)
	// }
	// tap, err := r.AttachPrivate("123", &net.IPNet{
	// 	IP:   net.ParseIP("192.168.1.15"),
	// 	Mask: net.CIDRMask(24, 32),
	// })

	// if err != nil {
	// 	return fmt.Errorf("failed to attach to private network: %w", err)
	// }
	// fmt.Println(tap)

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
