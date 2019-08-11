package main

import (
	"context"
	"flag"
	"os"

	"github.com/threefoldtech/zosv2/modules/stubs"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/threefoldtech/zosv2/modules/version"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		msgBrokerCon string
		resURL       string
		tnodbURL     string
		debug        bool
		ver          bool
		lruMaxSize   int
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&tnodbURL, "tnodb", "https://tnodb.dev.grid.tf", "address of tenant network object database")
	flag.StringVar(&resURL, "url", "https://tnodb.dev.grid.tf", "URL of the reservation server to poll from")
	flag.IntVar(&lruMaxSize, "cache", 10, "Number of reservation ID to keep in cache for owenership verification")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	if debug {
		log.Logger.Level(zerolog.DebugLevel)
	}

	flag.Parse()

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	identity := stubs.NewIdentityManagerStub(client)
	nodeID := identity.NodeID()

	cache := provision.NewCache(lruMaxSize, tnodbURL)

	// create context and add middlewares
	ctx := context.Background()
	ctx = provision.WithZBus(ctx, client)
	ctx = provision.WithTnoDB(ctx, tnodbURL)
	ctx = provision.WithCache(ctx, cache)

	// From here we start the real provision engine that will live
	// for the rest of the life of the node
	pipe, err := provision.FifoSource("/var/run/reservation.pipe")
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to allocation reservation pipe")
	}

	httpStore := provision.NewhHTTPStore(resURL)
	source := pipe
	if len(resURL) != 0 {
		source = provision.CompinedSource(
			pipe,
			provision.HTTPSource(httpStore, nodeID),
		)
	}

	engine := provision.New(source)

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	if err := engine.Run(ctx); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}
