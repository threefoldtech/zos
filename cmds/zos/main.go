package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	apigateway "github.com/threefoldtech/zos4/cmds/modules/api_gateway"
	"github.com/threefoldtech/zos4/cmds/modules/contd"
	"github.com/threefoldtech/zos4/cmds/modules/flistd"
	"github.com/threefoldtech/zos4/cmds/modules/netlightd"
	"github.com/threefoldtech/zos4/cmds/modules/noded"
	"github.com/threefoldtech/zos4/cmds/modules/powerd"
	"github.com/threefoldtech/zos4/cmds/modules/provisiond"
	"github.com/threefoldtech/zos4/cmds/modules/storaged"
	"github.com/threefoldtech/zos4/cmds/modules/vmd"
	"github.com/threefoldtech/zos4/cmds/modules/zbusdebug"
	"github.com/threefoldtech/zos4/cmds/modules/zui"
	"github.com/threefoldtech/zos4/pkg/app"
	"github.com/threefoldtech/zos4/pkg/version"
	"github.com/urfave/cli/v2"
)

func main() {
	app.Initialize()

	exe := cli.App{
		Name:    "ZOS",
		Version: version.Current().String(),

		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:   "list",
				Hidden: true,
				Usage:  "print all available clients names",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "force debug level",
			},
		},
		Commands: []*cli.Command{
			&zui.Module,
			&storaged.Module,
			&flistd.Module,
			&contd.Module,
			&vmd.Module,
			&noded.Module,
			&netlightd.Module,
			&provisiond.Module,
			&zbusdebug.Module,
			&powerd.Module,
			&apigateway.Module,
		},
		Before: func(c *cli.Context) error {
			if c.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Debug().Msg("setting log level to debug")
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			if !c.Bool("list") {
				cli.ShowAppHelpAndExit(c, 0)
			}
			// this hidden flag (complete) is used to list
			// all available modules names to automate building of
			// symlinks
			for _, cmd := range c.App.VisibleCommands() {
				if cmd.Name == "help" {
					continue
				}
				fmt.Println(cmd.Name)
			}
			return nil
		},
	}

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}
	name := filepath.Base(os.Args[0])
	args := os.Args
	for _, cmd := range exe.Commands {
		if cmd.Name == name {
			args = make([]string, 0, len(os.Args)+1)
			// this converts /bin/name <args> to  'zos <name> <args>
			args = append(args, "bin", name)
			args = append(args, os.Args[1:]...)
			break
		}
	}

	if err := exe.Run(args); err != nil {
		log.Fatal().Err(err).Msg("exiting")
	}
}
