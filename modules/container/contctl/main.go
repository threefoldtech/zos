package main

import (
	"flag"
	"os"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/stubs"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var (
		msgBrokerCon string
		flist        string
		name         string
		// netns        string
		entrypoint string
	)

	flag.StringVar(&msgBrokerCon, "broker", "tcp://localhost:6379", "connection string to the message broker")
	flag.StringVar(&flist, "flist", "", "URL to flist")
	flag.StringVar(&name, "name", "", "name of the container")
	// flag.StringVar(&netns, "netns", "netcont1", "network namespace name")
	flag.StringVar(&entrypoint, "entrypoint", "", "process to start in the container")
	flag.Parse()

	action := flag.Arg(0)

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Msgf("fail to connect to message broker client: %v", err)
	}

	c := stubs.NewContainerModuleStub(client)

	if action == "run" {
		data := modules.Container{
			FList: flist,
			Name:  name,
			Network: modules.NetworkInfo{
				Namespace: name,
			},
			Entrypoint: entrypoint,
		}
		log.Info().Msgf("start container with %+v", data)
		containerID, err := c.Run(name, data)
		if err != nil {
			log.Fatal().Err(err).Msgf("fail to create container %v", err)
			return
		}
		log.Info().Str("id", string(containerID)).Msg("container created")
	} else if action == "stop" {
		// // id =
		// if err := c.Delete(name, name); err != nil {
		// 	log.Fatal().Err(err).Msgf("fail to delete container %v", err)
		// 	return
		// }
		// log.Info().Msg("container created")
	} else {
		log.Info().Msgf("action %snot supported", action)
	}
}
