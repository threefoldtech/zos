package main

import (
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/urfave/cli"
)

func cmdsGenerateID(c *cli.Context) error {
	k, err := identity.GenerateKeyPair()
	if err != nil {
		return err
	}

	return identity.SerializeSeed(k, c.String("output"))
}
