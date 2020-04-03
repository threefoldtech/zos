package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/identity"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <original-seed> <converted-seed>\n", os.Args[0])
		os.Exit(1)
	}

	original := os.Args[1]
	destination := os.Args[2]

	log.Info().Str("source", original).Msg("loading original seed")

	// Load original seed file
	kp, err := identity.LoadKeyPair(original)
	if err != nil {
		log.Fatal().Err(err).Msg("load key pair")
	}

	// Create new object
	ud := &identity.UserData{
		Key:        kp,
		ThreebotID: 0,
	}

	// Save new object
	identity.SaveUserData(ud, destination)

	// Load new key to ensure loads works
	log.Info().Msg("reloading new seed to check")

	newkey, err := identity.LoadUserIdentity(destination)
	if err != nil {
		log.Fatal().Err(err).Msg("load user identity")
	}

	if bytes.Equal(newkey.Key.PrivateKey, kp.PrivateKey) {
		log.Info().Msg("keys matches")

	} else {
		log.Error().Msg("keys doesn't matches")
	}
}
