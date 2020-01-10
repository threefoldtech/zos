package main

import (
	"fmt"

	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/crypto"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/urfave/cli"
)

func encryptPassword(password, nodeID string) (string, error) {
	if len(password) == 0 {
		return "", nil
	}

	pubkey, err := crypto.KeyFromID(pkg.StrIdentifier(nodeID))
	if err != nil {
		return "", err
	}

	encrypted, err := crypto.Encrypt([]byte(password), pubkey)
	return base58.Encode(encrypted), err
}

func generateZDB(c *cli.Context) error {
	var (
		size     = c.Uint64("size")
		mode     = c.String("mode")
		password = c.String("password")
		nodeid   = c.String("node")
		disktype = c.String("type")
		public   = c.Bool("Public")
	)

	if len(password) != 0 && len(nodeid) == 0 {
		return fmt.Errorf("node ID is required if password is set")
	}

	if pkg.DeviceType(disktype) != pkg.HDDDevice && pkg.DeviceType(disktype) != pkg.SSDDevice {
		return fmt.Errorf("volume type can only 'HHD' or 'SSD'")
	}

	if mode != pkg.ZDBModeSeq && mode != pkg.ZDBModeUser {
		return fmt.Errorf("mode can only 'user' or 'seq'")
	}

	if size < 1 { //TODO: upper bound ?
		return fmt.Errorf("size cannot be less than 1")
	}

	encryptedPassword, err := encryptPassword(password, nodeid)
	zdb := provision.ZDB{
		Size:     size,
		DiskType: pkg.DeviceType(disktype),
		Mode:     pkg.ZDBMode(mode),
		Password: encryptedPassword,
		Public:   public,
	}

	p, err := embed(zdb, provision.ZDBReservation)
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), p)
}
