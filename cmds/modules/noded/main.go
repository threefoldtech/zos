package noded

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/monitord"
	"github.com/threefoldtech/zos/pkg/node"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
	"github.com/threefoldtech/zos/pkg/zinit"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

const (
	module          = "node"
	registrarModule = "registrar"
	eventsBlock     = "/tmp/events.chain"
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "noded",
	Usage: "reports the node total resources",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.BoolFlag{
			Name:  "id",
			Usage: "print node id and exit",
		},
		&cli.BoolFlag{
			Name:  "net",
			Usage: "print node network and exit",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		printID      bool   = cli.Bool("id")
		printNet     bool   = cli.Bool("net")
	)
	env := environment.MustGet()

	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	consumer, err := events.NewConsumer(msgBrokerCon, module)
	if err != nil {
		return errors.Wrap(err, "failed to to create event consumer")
	}

	if printID {
		sysCl := stubs.NewSystemMonitorStub(redis)
		fmt.Println(sysCl.NodeID(cli.Context))
		return nil
	}

	if printNet {
		fmt.Println(env.RunningMode.String())
		return nil
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})
	if err := registration(ctx, msgBrokerCon, env); err != nil {
		return errors.Wrap(err, "failed to start registration service")
	}

	if err := rmbApi(ctx, redis, msgBrokerCon); err != nil {
		return errors.Wrap(err, "failed to start node rmb api")
	}

	// block indefinietly, and other modules will get an error
	// when calling the registrar NodeID
	for app.CheckFlag(app.LimitedCache) {
		// logs are in the registrar
		time.Sleep(time.Minute * 5)
	}

	registrar := stubs.NewRegistrarStub(redis)
	var twin, nodeID uint32
	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Minute
	bo := backoff.WithContext(exp, ctx)
	err = backoff.RetryNotify(func() error {
		var err error
		nodeID, err = registrar.NodeID(ctx)
		if err != nil {
			return err
		}
		twin, err = registrar.TwinID(ctx)
		if err != nil {
			return err
		}
		return err
	}, bo, retryNotify)
	if err != nil {
		return errors.Wrap(err, "failed to get node id")
	}

	identity := stubs.NewIdentityManagerStub(redis)

	sk := ed25519.PrivateKey(identity.PrivateKey(ctx))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	log.Info().Str("address", id.Address()).Msg("node address")
	if err != nil {
		return err
	}

	sub, err := environment.GetSubstrate()
	if err != nil {
		return err
	}

	uptime, err := node.NewUptime(sub, id)
	if err != nil {
		return errors.Wrap(err, "failed to initialize uptime reported")
	}

	// start uptime reporting
	go uptime.Start(ctx)

	go func() {
		// wait for the uptime to be send before powering off the node
		if err := uptime.Mark.Done(ctx); err != nil {
			// context was cancelled but the uptime reporting was never done
			// the entire module is shutting down anyway
			log.Error().Err(err).Msg("failed waiting on the first uptime to be sent")
			return
		}

		applyPowerTarget(sub, nodeID)
	}()

	// node registration is completed we need to check the power target of the node.
	system, err := monitord.NewSystemMonitor(nodeID, 2*time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize system monitor")
	}

	host, err := monitord.NewHostMonitor(2 * time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize host monitor")
	}

	server.Register(zbus.ObjectID{Name: "host", Version: "0.0.1"}, host)
	server.Register(zbus.ObjectID{Name: "system", Version: "0.0.1"}, system)

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("unexpected error")
		}
	}()

	log.Info().Uint32("node", nodeID).Uint32("twin", twin).Msg("node registered")

	go func() {
		for {
			if err := public(ctx, node, env, redis, consumer); err != nil {
				log.Error().Err(err).Msg("setting public config failed")
				<-time.After(10 * time.Second)
			}
		}
	}()

	log.Info().Uint32("twin", twin).Msg("node has been registered")
	log.Debug().Msg("start message bus")
	return runMsgBus(ctx, sub, id)
}

func retryNotify(err error, d time.Duration) {
	// .Err() is scary (red)
	log.Warn().Str("err", err.Error()).Str("sleep", d.String()).Msg("the node isn't ready yet")
}

// check node target power status. Power off if need to be down
func applyPowerTarget(sub substrate.Manager, nodeID uint32) error {
	log.Info().Msg("checking power status of the node")

	client, err := sub.Substrate()
	if err != nil {
		return errors.Wrap(err, "failed to get connection to substrate")
	}
	defer client.Close()
	node, err := client.GetNode(nodeID)
	if err != nil {
		return errors.Wrap(err, "failed to get node information")
	}

	if !node.Power().IsDown {
		return nil
	}

	// is down!
	init := zinit.Default()
	err = init.Shutdown()

	if errors.Is(err, zinit.ErrNotSupported) {
		log.Info().Msg("node does not support shutdown. rebooting to update")
		return init.Reboot()
	}

	return err
}
