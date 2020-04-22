package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/tools/builders"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"

	"github.com/urfave/cli"
)

func generateZDB(c *cli.Context) error {
	var (
		size     = c.Int64("size")
		mode     = c.String("mode")
		password = c.String("password")
		disktype = c.String("type")
		public   = c.Bool("Public")
	)

	if disktype != workloads.DiskTypeHDD.String() && disktype != workloads.DiskTypeSSD.String() {
		return fmt.Errorf("volume type can only hdd or ssd")
	}

	if mode != workloads.ZDBModeSeq.String() && mode != workloads.ZDBModeUser.String() {
		return fmt.Errorf("mode can only 'user' or 'seq'")
	}

	if size < 1 { //TODO: upper bound ?
		return fmt.Errorf("size cannot be less than 1")
	}

	zdbBuilder := builders.NewZdbBuilder()
	zdbBuilder.WithSize(size).WithPassword(password).WithPublic(public)

	if mode == workloads.ZDBModeSeq.String() {
		zdbBuilder.WithMode(workloads.ZDBModeSeq)
	} else if mode == workloads.ZDBModeUser.String() {
		zdbBuilder.WithMode(workloads.ZDBModeUser)
	}

	if disktype == workloads.DiskTypeHDD.String() {
		zdbBuilder.WithDiskType(workloads.DiskTypeHDD)
	} else if disktype == workloads.DiskTypeSSD.String() {
		zdbBuilder.WithDiskType(workloads.DiskTypeSSD)
	}

	zdb, err := zdbBuilder.Build()
	if err != nil {
		return errors.Wrap(err, "failed to build to build zdb")
	}

	return writeWorkload(c.GlobalString("output"), zdb)
}
