package main

import (
	"fmt"
	"os"

	"github.com/cenkalti/backoff"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules/network"
	"github.com/threefoldtech/zosv2/modules/zinit"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := backoff.Retry(bootstrap, backoff.NewExponentialBackOff())
	if err != nil {
		return
	}

}

func bootstrap() error {
	log.Info().Msg("Start network bootstrap")
	if err := network.Bootstrap(); err != nil {
		return err
	}

	log.Info().Msg("writing udhcp init service")

	err := zinit.AddService("dhcp_zos", zinit.InitService{
		Exec:    fmt.Sprintf("/sbin/udhcpc -v -f -i %s -s /usr/share/udhcp/simple.script", network.DefaultBridgeName()),
		Oneshot: false,
		After:   []string{},
	})
	if err != nil {
		return err
	}

	return zinit.Monitor("dhcp_zos")
}
