package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/storage"
)

const (
	redisSocket = "unix:///var/run/redis.sock"
	module      = "storage"

	defaultRaid  = modules.Single // no raid
	defaultDisks = 1              // default 1 disk per volume (for no-raid setup)
	defaultPools = 0              // use as much disks as possible to create volumes
)

func main() {
	var (
		msgBrokerCon string
		workerNr     uint

		raid  string
		disks uint
		pools uint
	)

	flag.StringVar(&msgBrokerCon, "broker", redisSocket, "Connection string to the message broker")
	flag.UintVar(&workerNr, "workers", 1, "Number of workers")

	flag.StringVar(&raid, "raid", string(defaultRaid), "Raid setup to use for volumes")
	flag.UintVar(&disks, "disks", defaultDisks, "Number of disks per volume")
	flag.UintVar(&pools, "volumes", defaultPools, "Amount of volumes to try and create")

	flag.Parse()

	storage := storage.New()
	policy := modules.StoragePolicy{
		Raid:     modules.RaidProfile(raid),
		Disks:    uint8(disks),
		MaxPools: uint8(pools),
	}
	if err := storage.Initialize(policy); err != nil {
		log.Fatal().Msgf("Failed to initialize storage: %s", err)
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, storage)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting storaged module")

	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}

	log.Warn().Msgf("Exiting storaged")
}
