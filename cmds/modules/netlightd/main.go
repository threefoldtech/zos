package netlightd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/oasisprotocol/curve25519-voi/primitives/x25519"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/netlight"
	"github.com/threefoldtech/zos/pkg/netlight/resource"
	"github.com/urfave/cli/v2"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	module      = "netlight"
)

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
	},
	Action: action,
}

func myceliumSeedFromIdentity(privKey []byte) []byte {
	seed := x25519.PrivateKey(x25519.EdPrivateKeyToX25519([]byte(privKey)))
	return seed[:]
}

func action(cli *cli.Context) error {
	var (
		root   string = cli.String("root")
		broker string = cli.String("broker")
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

	identity := stubs.NewIdentityManagerStub(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := ensureHostFw(ctx); err != nil {
		return errors.Wrap(err, "failed to host firewall rules")
	}

	bridge, err := netlight.CreateNDMZBridge()
	if err != nil {
		return fmt.Errorf("failed to create ndmz bridge: %w", err)
	}

	err = resource.Create("dmz", bridge, &net.IPNet{
		IP:   net.ParseIP("100.127.0.2"),
		Mask: net.CIDRMask(16, 32),
	}, netlight.NDMZGwIP, nil, myceliumSeedFromIdentity(identity.PrivateKey(cli.Context)))

	if err != nil {
		return fmt.Errorf("failed to create ndmz resource: %w", err)
	}

	// create a test user network
	err = resource.Create("test", bridge, &net.IPNet{
		IP:   net.ParseIP("100.127.0.10"),
		Mask: net.CIDRMask(16, 32),
	}, netlight.NDMZGwIP, &net.IPNet{
		IP:   net.ParseIP("192.168.1.0"),
		Mask: net.CIDRMask(24, 32),
	}, zos.MustBytesFromHex("8ad7d29b81df3f3ef0a5ff95c25cc0824ef33137fbbcf22d2f23b0222ae3ac00"))

	if err != nil {
		return fmt.Errorf("failed to create user resource: %w", err)
	}

	select {}
	//return nil
}

func waitMyceliumBin() {
	log.Info().Msg("wait for yggdrasil binary to be available")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // forever
	_ = backoff.RetryNotify(func() error {
		_, err := exec.LookPath("mycelium")
		return err
	}, bo, func(err error, d time.Duration) {
		log.Warn().Err(err).Msgf("yggdrasil binary not found, retying in %s", d.String())
	})
}
