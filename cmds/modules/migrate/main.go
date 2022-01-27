package migrate

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/urfave/cli/v2"

	"github.com/rs/zerolog/log"
)

const (
	LongWait  = 3 * time.Hour
	ShortWait = 30 * time.Minute
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:   "migrate",
	Usage:  "migrates nodes to v3",
	Action: action,
}

type Farm struct {
	ID      uint64 `json:"farmID"`
	Name    string `json:"name"`
	Address string `json:"stellarAddress"`
}

// farmsWithWallet returns v3 farms that has the given pay address
func farmsWithWallet(ctx context.Context, url, wallet string) ([]Farm, error) {
	cl := graphql.NewClient(url)

	req := graphql.NewRequest(`
    query ($address: String!) {
		farms(where: {stellarAddress_eq: $address}) {
		  name
		  farmId
		  stellarAddress
		}
	  }

	`)

	req.Var("address", wallet)
	var response struct {
		Farms []Farm `json:"farms"`
	}

	if err := cl.Run(ctx, req, &response); err != nil {
		return nil, errors.Wrap(err, "failed to query farms with payment address")
	}

	return response.Farms, nil
}

func walletAddress(farm *directory.Farm) string {
	const ASSET = "TFT"

	for _, addr := range farm.WalletAddresses {
		if addr.Asset == ASSET {
			return addr.Address
		}
	}

	return ""
}

func migrationFarm(candidates []Farm, name string) *Farm {
	if len(candidates) == 0 {
		return nil
	} else if len(candidates) == 1 {
		return &candidates[0]
	}

	// other wise
	for i := range candidates {
		farm := &candidates[i]
		if strings.EqualFold(farm.Name, name) {
			return farm
		}
	}
	// just choose the first one
	return &candidates[0]
}

func migrate(ctx context.Context, v2 *directory.Farm, v3 *Farm) error {
	// we finally know which farms we need to migrate from->to. we need
	// to do the following:
	// - first of all, find the usb stick, if it's not there, nothing we can do
	// - stop provisiond + storaged
	// - rewrite the usb stick with new v3 image with correct farm id
	// - wipe the disks
	// - sync
	// - reboot
	// - cross fingers

	sticks, err := devices(ctx, IsUsb)
	if err != nil {
		return errors.Wrap(err, "failed to list usb sticks")
	}

	if len(sticks) == 0 {
		// it's better if we error here so we don't proceed with update procedure
		return fmt.Errorf("no usb sticks detected")
	}

	cl, err := zinit.New("/var/run/zinit.sock")
	if err != nil {
		return errors.Wrap(err, "failed to connect to zinit")
	}
	if err := cl.StopMultiple(2*time.Minute, "storaged", "provisiond"); err != nil {
		return errors.Wrap(err, "Failed to stop services")
	}

	// if there are multiple usb sticks, we can't tell which one to use
	// so we do all. good luck to the user
	for _, usb := range sticks {
		log.Info().Str("usb", usb.Path).Msg("rewriting usb stick")
		if err := burn(ctx, v3.ID, &usb); err != nil {
			return nil
		}
	}

	syscall.Sync()

	if err := wipe(ctx); err != nil {
		return errors.Wrap(err, "failed to wipe disks")
	}

	syscall.Sync()

	if err := crossFingers(); err != nil {
		return errors.Wrap(err, "failed to reboot")
	}
	log.Info().Msg("I should never show up on screen")
	return nil
}

func crossFingers() error {
	f, err := os.OpenFile("/proc/sysrq-trigger", os.O_WRONLY, 0400)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("b")
	return err
}

func action(cli *cli.Context) error {
	app.Initialize()
	log.Info().Msg("starting upgrade daemon")

	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get environment")
	}

	// address := "GCE4MAASAFI3AT3U7CCDJ5OGZGWNUQ2CE2Q6V3HVNKEF2UJI3RPWPTWG"

	cl, err := client.NewClient(env.BcdbURL, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create explorer client")
	}

	v2, err := cl.Directory.FarmGet(schema.ID(env.FarmerID))
	address := walletAddress(&v2)

	if len(address) == 0 {
		// we check again in longer period because of this farm doesn't have a payout until
		// today so what the ** this farmer was doing all this time.
		log.Info().Msgf("farm is not associated with a payout address. We will check again in %s", LongWait)
		<-time.After(LongWait)
		return nil
	}

	matches, err := farmsWithWallet(cli.Context, env.GraphQlURL, address)

	if err != nil {
		return errors.Wrapf(err, "failed to get v3 farms with payout address '%s'", address)
	}

	v3 := migrationFarm(matches, v2.Name)

	if v3 == nil {
		// there are no farms configured on v3 that has the same payout address.
		// we try again sooner.
		log.Info().Dur("wait", ShortWait).Str("wallet", address).Msg("no farms found with this payout address on v3. retrying again after wait")
		<-time.After(ShortWait)
		return nil
	}

	log.Info().Msgf("migration from V2(%d - %s) -> V3(%d - %s)", v2.ID, v2.Name, v3.ID, v3.Name)

	bf := backoff.NewExponentialBackOff()
	bf.InitialInterval = ShortWait
	bf.MaxInterval = time.Hour
	backoff.RetryNotify(func() error {
		return migrate(cli.Context, &v2, v3)
	}, bf, func(e error, d time.Duration) {
		log.Error().Err(err).Dur("wait", d).Msg("migration failed, retrying after wait")
	})

	return nil
}
