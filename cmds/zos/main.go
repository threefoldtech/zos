package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/threefoldtech/zos/cmds/modules/contd"
	"github.com/threefoldtech/zos/cmds/modules/flistd"
	"github.com/threefoldtech/zos/cmds/modules/storaged"
	"github.com/threefoldtech/zos/cmds/modules/vmd"
	"github.com/threefoldtech/zos/cmds/modules/zbusdebug"
	"github.com/threefoldtech/zos/cmds/modules/zui"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/version"
	cli "github.com/urfave/cli/v2"
)

func main() {
	app.Initialize()

	exe := cli.App{
		Name:    "ZOS",
		Version: version.Current().String(),
		Commands: []*cli.Command{
			&zui.Module,
			&storaged.Module,
			&flistd.Module,
			&contd.Module,
			&vmd.Module,
			&zbusdebug.Module,
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
