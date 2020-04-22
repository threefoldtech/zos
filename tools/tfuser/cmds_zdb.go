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

	var zdbMode workloads.ZDBModeEnum
	if mode == workloads.ZDBModeSeq.String() {
		zdbMode = workloads.ZDBModeSeq
	} else if mode == workloads.ZDBModeUser.String() {
		zdbMode = workloads.ZDBModeUser
	}

	var zdbDiskType workloads.DiskTypeEnum
	if disktype == workloads.DiskTypeHDD.String() {
		zdbDiskType = workloads.DiskTypeHDD
	} else if disktype == workloads.DiskTypeSSD.String() {
		zdbDiskType = workloads.DiskTypeSSD
	}

	zdbBuilder := builders.NewZdbBuilder(c.String("node"), size, zdbMode, zdbDiskType)
	zdbBuilder.WithPassword(password).WithPublic(public)

	zdb, err := zdbBuilder.Build()
	if err != nil {
		return errors.Wrap(err, "failed to build to build zdb")
	}

	return writeWorkload(c.GlobalString("output"), zdb)
}
