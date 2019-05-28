package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/container"
)

var (
	msgBrokerCon  = flag.String("broker", "tcp://localhost:6379", "connection string to the message broker")
	containerdCon = flag.String("containerd", "/run/containerd/containerd.sock", "connection string to containerd")
	workerNr      = flag.Uint("workers", 1, "number of workers")
)

const module = "container"

func main() {
	flag.Parse()

	server, err := zbus.NewRedisServer(module, *msgBrokerCon, *workerNr)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v", err)
	}

	client, err := zbus.NewRedisClient(*msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker client: %v", err)
	}

	containerd := container.New(client, *containerdCon)

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, containerd)

	log.Info().
		Str("broker", *msgBrokerCon).
		Uint("worker nr", *workerNr).
		Msg("starting containerd module")

	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}
