package noded

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
)

const (
	redisAddr = "unix:///var/run/redis.sock"
	keyType   = "ed25519"
)

func runMsgBus(ctx context.Context, sk ed25519.PrivateKey, substrateURLs []string) error {
	// select the first one as only one URL is set for now
	if len(substrateURLs) == 0 {
		return errors.New("at least one substrate URL must be provided")
	}

	seed := sk.Seed()
	seedHex := fmt.Sprintf("0x%s", hex.EncodeToString(seed))

	log.Info().Msg("starting rmb...")
	command := exec.CommandContext(ctx, "rmb", "-s", substrateURLs[0], "-k", keyType, "--seed", seedHex, "-r", redisAddr)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
