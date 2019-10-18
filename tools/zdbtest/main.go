package main

import (
	"flag"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/storage"
)

const (
	MiB = 1024 * 1024
	GiB = 1024 * MiB
)

func main() {
	////log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	var (
		size       uint64
		deviceType string
		claim      bool
		name       string
	)

	flag.Uint64Var(&size, "size", 1, "size to allocation in GiB")
	flag.StringVar(&deviceType, "type", "SSD", "type of device")
	flag.BoolVar(&claim, "claim", false, "claim")
	flag.StringVar(&name, "name", "", "name of volume to claim")
	flag.Parse()

	storage, err := storage.New()
	if err != nil {
		log.Fatal().Msgf("Error initializing storage module: %s", err)
	}

	if !claim {
		name, path, err := storage.Allocate(pkg.DeviceType(deviceType), size*GiB, pkg.ZDBModeSeq)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to allocate space for zdb")
		}

		log.Info().
			Str("name", name).
			Str("path", path).
			Msgf("allocation done")
	} else {
		err := storage.Claim(name, size*GiB)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to claim")
		}
		log.Info().
			Str("name", name).
			Msgf("storage claimed")
	}
}
