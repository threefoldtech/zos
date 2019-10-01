package main

import (
	"fmt"
	"strings"

	"github.com/threefoldtech/zosv2/pkg"

	"github.com/threefoldtech/zosv2/pkg/provision"
	"github.com/urfave/cli"
)

func generateVolume(c *cli.Context) error {
	s := c.Uint64("size")
	t := strings.ToUpper(c.String("type"))

	if t != pkg.HDDDevice && t != pkg.SSDDevice {
		return fmt.Errorf("volume type can only HHD or SSD")
	}

	if s < 1 { //TODO: upper bound ?
		return fmt.Errorf("size cannot be less then 1")
	}

	v := provision.Volume{
		Size: s,
		Type: provision.DiskType(t),
	}

	p, err := embed(v, provision.VolumeReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), p)
}
