package main

import (
	"context"
	"flag"
	"os"
	"path"

	"github.com/threefoldtech/zbus"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/gedis"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/version"
)

const (
	seedName = "seed.txt"
	module   = "identityd"
	workers  = 10
)

func main() {
	var (
		root         string
		msgBrokerCon string
		bcdbAddr     string
		bcdbNs       string
		bcdbPass     string
		ver          bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/identityd", "root working directory of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.StringVar(&bcdbAddr, "bcdbaddr", "", "address of the blockchain database")
	flag.StringVar(&bcdbNs, "bcdbns", "default", "namespace inside the blockchain database")
	flag.StringVar(&bcdbPass, "bcdbpass", "", "password of the namespace blockchain database")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := os.MkdirAll(root, 0755); err != nil {
		log.Fatal().Err(err).Msg("failed to create module root")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workers)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create zbus server")
	}

	manager, err := identity.NewManager(path.Join(root, seedName))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create identity manager")
	}

	farmID, err := manager.FarmID()
	if err != nil {
		log.Fatal().Err(err).Msg("fail to read farmer id from kernel parameters")
	}

	nodeID := manager.NodeID()
	log.Info().
		Str("identity", nodeID.Identity()).
		Msg("node identity loaded")

	// Node registration can happen in the background.
	go func() {
		// store := identity.NewHTTPIDStore(tnodbURL)
		store, err := gedis.New(bcdbAddr, bcdbNs, bcdbPass)
		if err != nil {
			log.Error().Err(err).Msg("fail to connect to blockchain")
			return
		}

		f := func() error {
			log.Info().Msg("start registration of the node")
			_, err := store.RegisterNode(nodeID, farmID)
			if err != nil {
				log.Error().Err(err).Msg("fail to register node identity")
				return err
			}
			return nil
		}

		if err := backoff.Retry(f, backoff.NewExponentialBackOff()); err == nil {
			log.Info().Msg("node registered successfully")
		}
	}()

	if err := server.Register(zbus.ObjectID{
		Name:    "manager",
		Version: "0.0.1",
	}, manager); err != nil {
		log.Fatal().Err(err).Msg("failed to register identity manager")
	}

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting identity module")
	if err := server.Run(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("server exit")
	}
}
