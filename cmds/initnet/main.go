package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/network/ifaceutil"

	"github.com/threefoldtech/zosv2/modules/version"
	"github.com/threefoldtech/zosv2/modules/zinit"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "initnet"

func main() {
	var (
		root   string
		broker string
		ver    bool
	)

	flag.StringVar(&root, "root", "/var/cache/modules/initnet", "root path of the module")
	flag.StringVar(&broker, "broker", redisSocket, "connection string to broker")
	flag.BoolVar(&ver, "v", false, "show version and exit")
	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := os.MkdirAll(root, 0750); err != nil {
		log.Error().Err(err).Msgf("fail to create module root")
	}
	if err := ifaceutil.SetLoUp(); err != nil {
		return
	}
	if err := bootstrap(); err != nil {
		log.Error().Err(err).Msg("failed to bootstrap network")
		os.Exit(1)
	}

	log.Info().Msg("network bootstrapped successfully")

	if err := ready(); err != nil {
		log.Fatal().Err(err).Msg("failed to mark networkd as ready")
	}
}

func ready() error {
	f, err := os.Create("/var/run/initnet.ready")
	defer f.Close()
	return err
}

func bootstrap() error {
	f := func() error {

		z := zinit.New("")
		if err := z.Connect(); err != nil {
			log.Error().Err(err).Msg("failed to connect to zinit")
			return err
		}

		log.Info().Msg("Start network bootstrap")
		if err := network.Bootstrap(); err != nil {
			log.Error().Err(err).Msg("fail to boostrap network")
			return err
		}

		log.Info().Msg("writing udhcp init service")

		err := zinit.AddService("dhcp_zos", zinit.InitService{
			Exec:    fmt.Sprintf("dhcpcd %s", network.DefaultBridge),
			Oneshot: false,
			After:   []string{},
		})
		if err != nil {
			log.Error().Err(err).Msg("fail to create dhcp_zos zinit service")
			return err
		}

		if err := z.Monitor("dhcp_zos"); err != nil {
			log.Error().Err(err).Msg("fail to start monitoring dhcp_zos zinit service")
			return err
		}
		return nil
	}

	errHandler := func(err error, t time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while trying to bootstrap network")
		}
	}

	return backoff.RetryNotify(f, backoff.NewExponentialBackOff(), errHandler)
}
