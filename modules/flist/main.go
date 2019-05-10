package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
)

var (
	moduleRoot = flag.String("root", defaultRoot, "root working directory of the module")
)

const module = "flist"

func main() {
	flag.Parse()

	server, err := zbus.NewRedisServer(module, "tcp://localhost:6379", 1)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker server: %v\n", err)
	}

	flist := New(*moduleRoot)
	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, flist)
	log.Info().Msg("starting flist module")
	if err := server.Run(context.Background()); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
}
