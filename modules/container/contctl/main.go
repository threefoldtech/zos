package main

import (
	"os"

	"github.com/threefoldtech/zosv2/modules"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/network/namespace"
	"github.com/threefoldtech/zosv2/modules/network/wireguard"
	"github.com/threefoldtech/zosv2/modules/stubs"
	"github.com/urfave/cli"
)

func main() {
	var (
		client    zbus.Client
		container *stubs.ContainerModuleStub
		flistd    *stubs.FlisterStub
	)
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "broker",
			Value: "unix:///var/run/redis.sock",
		},
	}
	app.Before = func(c *cli.Context) error {
		broker := c.String("broker")

		cl, err := zbus.NewRedisClient(broker)
		if err != nil {
			log.Error().Msgf("fail to connect to message broker client: %v", err)
			return err
		}
		client = cl
		container = stubs.NewContainerModuleStub(client)
		flistd = stubs.NewFlisterStub(client)
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "start a container",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "flist",
					Value: "",
				},
				cli.StringFlag{
					Name:  "name",
					Value: "",
				},
				cli.StringFlag{
					Name:  "entrypoint",
					Value: "",
				},
				cli.BoolFlag{
					Name:  "interactive",
					Usage: "Enable webui console",
				},
			},
			Before: func(c *cli.Context) error {
				log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
				return nil
			},
			Action: func(c *cli.Context) error {

				flist := c.String("flist")
				name := c.String("name")
				entrypoint := c.String("entrypoint")
				interactive := c.Bool("interactive")

				// create wg interface in host net namespace
				wg, err := wireguard.New("wg0")
				if err != nil {
					return err
				}

				// create container net namespace
				_, err = namespace.CreateNetNS(name)
				if err != nil {
					return err
				}

				// enter container net ns
				nsCtx := namespace.NSContext{}
				nsCtx.Enter(name)

				// move wg iface into container netns
				if err := namespace.SetLinkNS(wg, name); err != nil {
					nsCtx.Exit()
					return err
				}

				// configure wg iface
				err = wg.Configure("172.21.0.10/24", "2MDD+PDklXfOd+1jRWXE/aIwVurvbI6I7I10KBaNvHg=", []wireguard.Peer{
					{
						PublicKey:  "mR5fBXohKe2MZ6v+GLwlKwrvkFxo1VvV3bPNHDBhOAI=",
						Endpoint:   "37.187.124.71:51820",
						AllowedIPs: []string{"0.0.0.0/0"},
					},
				})
				if err != nil {
					nsCtx.Exit()
					return err
				}

				// exit containe net ns
				nsCtx.Exit()

				rootfs, err := flistd.Mount(flist, "")
				if err != nil {
					return err
				}

				data := modules.Container{
					RootFS: rootfs,
					Name:   name,
					Network: modules.NetworkInfo{
						Namespace: name,
					},
					Entrypoint:  entrypoint,
					Interactive: interactive,
				}
				log.Info().Msgf("start container with %+v", data)
				containerID, err := container.Run(name, data)
				if err != nil {
					log.Error().Err(err).Msgf("fail to create container %v", err)
					return err
				}
				log.Info().Str("id", string(containerID)).Msg("container created")
				return nil
			},
		},
		{
			Name:  "stop",
			Usage: "stop a container",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "name",
					Value: "",
				},
			},
			Before: func(c *cli.Context) error {
				log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
				return nil
			},
			Action: func(c *cli.Context) error {
				name := c.String("name")

				info, err := container.Inspect(name, modules.ContainerID(name))
				if err != nil {
					return err
				}

				if err := container.Delete(name, modules.ContainerID(name)); err != nil {
					log.Error().Err(err).Msgf("fail to delete container %v", err)
					return err
				}

				if err := flistd.Umount(info.RootFS); err != nil {
					return err
				}
				log.Info().Msg("container created")
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}
