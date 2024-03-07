package noded

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/monitord"
	"github.com/threefoldtech/zos/pkg/perf"
	"github.com/threefoldtech/zos/pkg/perf/cpubench"
	"github.com/threefoldtech/zos/pkg/perf/healthcheck"
	"github.com/threefoldtech/zos/pkg/perf/iperf"
	"github.com/threefoldtech/zos/pkg/perf/publicip"
	"github.com/threefoldtech/zos/pkg/registrar"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

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

func registerationServer(ctx context.Context, msgBrokerCon string, env environment.Environment, info registrar.RegistrationInfo) error {

	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	server, err := zbus.NewRedisServer(registrarModule, msgBrokerCon, 1)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	registrar := registrar.NewRegistrar(ctx, redis, env, info)
	server.Register(zbus.ObjectID{Name: "registrar", Version: "0.0.1"}, registrar)
	log.Debug().Msg("object registered")
	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("unexpected error exited registrar")
	}
	return nil
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

	oracle := capacity.NewResourceOracle(stubs.NewStorageModuleStub(redis))
	cap, err := oracle.Total()
	if err != nil {
		return errors.Wrap(err, "failed to get node capacity")
	}
	secureBoot, err := capacity.IsSecureBoot()
	if err != nil {
		log.Error().Err(err).Msg("failed to detect secure boot flags")
	}

	dmi, err := oracle.DMI()
	if err != nil {
		return errors.Wrap(err, "failed to get dmi information")
	}

	hypervisor, err := oracle.GetHypervisor()
	if err != nil {
		return errors.Wrap(err, "failed to get hypervisors")
	}
	gpus, err := oracle.GPUs()
	if err != nil {
		return errors.Wrap(err, "failed to list gpus")
	}

	var info registrar.RegistrationInfo
	for _, gpu := range gpus {
		// log info about the GPU here ?
		vendor, device, ok := gpu.GetDevice()
		if ok {
			log.Info().Str("vendor", vendor.Name).Str("device", device.Name).Msg("found GPU")
		} else {
			log.Info().Uint16("vendor", gpu.Vendor).Uint16("device", device.ID).Msg("found GPU (can't look up device name)")
		}

		info = info.WithGPU(gpu.ShortID())
	}

	info = info.WithCapacity(cap).
		WithSerialNumber(dmi.BoardVersion()).
		WithSecureBoot(secureBoot).
		WithVirtualized(len(hypervisor) != 0)

	go registerationServer(ctx, msgBrokerCon, env, info)
	log.Info().Msg("start perf scheduler")

	perfMon, err := perf.NewPerformanceMonitor(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to create a new perfMon")
	}

	healthcheck.RunNTPCheck(ctx)
	perfMon.AddTask(iperf.NewTask())
	perfMon.AddTask(cpubench.NewTask())
	perfMon.AddTask(publicip.NewTask())
	perfMon.AddTask(healthcheck.NewTask())

	if err = perfMon.Run(ctx); err != nil {
		return errors.Wrap(err, "failed to run the scheduler")
	}

	// block indefinietly, and other modules will get an error
	// when calling the registrar NodeID
	if app.CheckFlag(app.LimitedCache) {
		for app.CheckFlag(app.LimitedCache) {
			// logs are in the registrar
			time.Sleep(time.Minute * 5)
		}
	}
	registrar := stubs.NewRegistrarStub(redis)
	var twin, node uint32
	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 2 * time.Minute
	bo := backoff.WithContext(exp, ctx)
	err = backoff.RetryNotify(func() error {
		var err error
		node, err = registrar.NodeID(ctx)
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
	sub, err := environment.GetSubstrate()
	if err != nil {
		return err
	}

	events, err := events.NewRedisStream(sub, msgBrokerCon, env.FarmID, node, eventsBlock)
	if err != nil {
		return err
	}
	go events.Start(ctx)

	system, err := monitord.NewSystemMonitor(node, 2*time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize system monitor")
	}

	host, err := monitord.NewHostMonitor(2 * time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize host monitor")
	}

	server.Register(zbus.ObjectID{Name: "host", Version: "0.0.1"}, host)
	server.Register(zbus.ObjectID{Name: "system", Version: "0.0.1"}, system)
	server.Register(zbus.ObjectID{Name: "host", Version: "0.0.1"}, perfMon)

	log.Info().Uint32("node", node).Uint32("twin", twin).Msg("node registered")

	go func() {
		for {
			if err := public(ctx, node, env, redis, consumer); err != nil {
				log.Error().Err(err).Msg("setting public config failed")
				<-time.After(10 * time.Second)
			}
		}
	}()

	log.Info().Uint32("twin", twin).Msg("node has been registered")
	idStub := stubs.NewIdentityManagerStub(redis)
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sk := ed25519.PrivateKey(idStub.PrivateKey(fetchCtx))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	log.Info().Str("address", id.Address()).Msg("node address")
	if err != nil {
		return err
	}
	if err := server.Run(ctx); err != nil && err != context.Canceled {
		log.Fatal().Err(err).Msg("unexpected error")
	}
	return nil
}

func retryNotify(err error, d time.Duration) {
	// .Err() is scary (red)
	log.Warn().Str("err", err.Error()).Str("sleep", d.String()).Msg("the node isn't ready yet")
}
