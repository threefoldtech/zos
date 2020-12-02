package provisiond

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/provision/explorer"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
	"github.com/threefoldtech/zos/pkg/provision/primitives/cache"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
)

const (
	module = "provision"
	gib    = 1024 * 1024 * 1024
)

// Module entry point
var Module cli.Command = cli.Command{
	Name:  "provisiond",
	Usage: "handles reservations streams and use other daemon to deploy them",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/provisiond",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		storageDir   string = cli.String("root")
	)

	flag.StringVar(&storageDir, "root", "/var/cache/modules/provisiond", "root path of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")

	// keep checking if limited-cache flag is set
	if app.CheckFlag(app.LimitedCache) {
		log.Error().Msg("failed cache reservation! Retrying every 30 seconds...")
		for app.CheckFlag(app.LimitedCache) {
			time.Sleep(time.Second * 30)
		}
	}

	if err := os.MkdirAll(storageDir, 0770); err != nil {
		return errors.Wrap(err, "failed to create cache directory")
	}

	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to parse node environment")
	}

	if env.Orphan {
		// disable providiond on this node
		// we don't have a valid farmer id set
		log.Info().Msg("orphan node, we won't provision anything at all")
		select {}
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		return errors.Wrap(err, "failed to connect to message broker")
	}
	zbusCl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker server")
	}

	identity := stubs.NewIdentityManagerStub(zbusCl)
	nodeID := identity.NodeID()

	// block until networkd is ready to serve request from zbus
	// this is used to prevent uptime and online status to the explorer if the node is not in a fully ready
	// https://github.com/threefoldtech/zos/issues/632
	network := stubs.NewNetworkerStub(zbusCl)
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	backoff.RetryNotify(func() error {
		return network.Ready()
	}, bo, func(err error, d time.Duration) {
		log.Error().Err(err).Msg("networkd is not ready yet")
	})

	// to get reservation from tnodb
	cl, err := app.ExplorerClient()
	if err != nil {
		return errors.Wrap(err, "failed to instantiate BCDB client")
	}

	// keep track of resource units reserved and amount of workloads provisionned

	// to store reservation locally on the node
	store, err := cache.NewFSStore(filepath.Join(storageDir, "reservations"))
	if err != nil {
		return errors.Wrap(err, "failed to create local reservation store")
	}

	const daemonBootFlag = "provisiond"
	if app.IsFirstBoot(daemonBootFlag) {
		if err := store.Purge(cache.NotPersisted); err != nil {
			log.Fatal().Err(err).Msg("failed to clean up cache")
		}
	}

	if err := app.MarkBooted(daemonBootFlag); err != nil {
		log.Fatal().Err(err).Msg("failed to mark service as booted")
	}

	// compatability fix
	if err := UpdateReservationsResults(store); err != nil {
		log.Fatal().Err(err).Msg("failed to upgrade cached reservations")
	}

	initial, err := store.CurrentCounters()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get current deployed capacity")
	}
	// update initial capacity with
	reserved, err := getNodeReserved(zbusCl)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get node reserved capacity")
	}

	handlers := primitives.NewPrimitivesProvisioner(zbusCl)
	/* --- committer
	 *   --- cache
	 *	   --- statistics
	 *	     --- handlers
	 */
	provisioner := explorer.NewCommitterProvisioner(
		provision.NewCachedProvisioner(
			primitives.NewStatisticsProvisioner(
				handlers,
				initial,
				reserved,
				nodeID.Identity(),
				cl.Directory,
			),
			store,
		),
		cl.Workloads,
		primitives.ResultToSchemaType,
		identity,
		nodeID.Identity(),
	)

	puller := explorer.NewPoller(cl.Workloads, primitives.WorkloadToProvisionType, primitives.ProvisionOrder)
	engine := provision.New(provision.EngineOps{
		Source: provision.CombinedSource(
			provision.PollSource(puller, nodeID),
			provision.NewDecommissionSource(store),
		),
		Provisioner: provisioner,
		Janitor:     primitives.NewJanitor(zbusCl, puller),
	})

	if err != nil {
		return errors.Wrap(err, "failed to instantiate provision engine")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, pkg.Provision(engine))

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	ctx := context.Background()
	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	// call the runtime upgrade before running engine
	handlers.RuntimeUpgrade(ctx)

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("unexpected error")
		}
		log.Info().Msg("zbus server stopped")
	}()

	if err := engine.Run(ctx); err != nil {
		return errors.Wrap(err, "unexpected error")
	}

	log.Info().Msg("provision engine stopped")
	return nil
}

func getNodeReserved(cl zbus.Client) (counter primitives.Counters, err error) {
	storage := stubs.NewStorageModuleStub(cl)
	fs, err := storage.GetCacheFS()
	if err != nil {
		return counter, err
	}

	var v *primitives.AtomicValue
	switch fs.DiskType {
	case pkg.HDDDevice:
		v = &counter.HRU
	case pkg.SSDDevice:
		v = &counter.SRU
	default:
		return counter, fmt.Errorf("unknown cache disk type '%s'", fs.DiskType)
	}

	v.Increment(fs.Usage.Size)
	counter.MRU.Increment(2 * gib)
	return
}
