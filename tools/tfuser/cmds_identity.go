package main

import (
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/phonebook"
	"github.com/urfave/cli"
)

func cmdsGenerateID(c *cli.Context) error {

	output := c.String("output")
	name := c.String("name")
	email := c.String("email")
	description := c.String("description")

	k, err := identity.LoadKeyPair(output)
	if err == nil {
		fmt.Printf("A seed already exists at %s\n", output)
		fmt.Printf("Identity: %s\n", k.Identity())
		return nil
	}

	k, err = identity.GenerateKeyPair()
	if err != nil {
		return err
	}

	user := phonebook.User{
		Name:        name,
		Email:       email,
		Pubkey:      hex.EncodeToString(k.PublicKey),
		Description: description,
	}

	if err := k.Save(c.String("output")); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}

	log.Debug().Str("bcdb", bcdbAddr).Str("output", c.String("output")).Msg("connecting")
	bcdb, err = getClient(bcdbAddr, c.String("output"))

	log.Debug().Msg("register user")
	id, err := bcdb.Phonebook.Create(user)
	if err != nil {
		return errors.Wrap(err, "failed to register user")
	}

	fmt.Printf("Your ID is: %d\n", id)
	fmt.Printf("Seed saved to: %s\n", output)
	return nil
}
