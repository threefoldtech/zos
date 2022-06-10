package noded

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/zinit"
	"time"
)

const (
	busService = "rmb"
	KeyType    = "ed25519"
)

func runMsgBus(ctx context.Context, sk ed25519.PrivateKey, substrateURLs []string) error {
	// select the first one as only one URL is set for now
	if len(substrateURLs) == 0 {
		return errors.New("at least one substrate URL must be provided")
	}

	seed := sk.Seed()
	seedHex := fmt.Sprintf("0x%s", hex.EncodeToString(seed))
	cmd := fmt.Sprintf(`/bin/rmb --substrate "%s" --key-type "%s" --seed "%s"`, substrateURLs[0], KeyType, seedHex)

	// just for debugging for now
	log.Info().Str("cmd", cmd).Msg("starting rmb with")

	cl := zinit.Default()
	err := zinit.AddService(busService, zinit.InitService{
		Exec: cmd,
	})

	if err != nil {
		return err
	}

	if err = cl.Monitor(busService); err != nil && !errors.Is(err, zinit.ErrAlreadyMonitored) {
		return err
	}

	return cl.StartWait(time.Second*20, busService)
}
