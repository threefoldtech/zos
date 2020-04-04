package main

import (
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/identity"
	"github.com/threefoldtech/zos/tools/client"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/phonebook"
	"github.com/urfave/cli"
)

func cmdsGenerateID(c *cli.Context) error {
	output := c.String("output")
	name := c.String("name")
	email := c.String("email")
	description := c.String("description")
	mnemonic := c.String("mnemonic")

	ui := identity.UserIdentity{}

	// Try to load the destination file (not allow overwrite)
	err := ui.Load(output)
	if err == nil {
		fmt.Printf("A seed already exists at: %s\n", output)
		fmt.Printf("Identity: %s\n", ui.Key().Identity())
		fmt.Printf("Threebot ID: %d\n", ui.ThreebotID)
		return nil
	}

	if mnemonic != "" {
		log.Info().Msg("building key using existing mnemonic")

		ui.Mnemonic = mnemonic
		err = ui.Initialize()
		if err != nil {
			return err
		}

	} else {
		k, err := identity.GenerateKeyPair()
		if err != nil {
			return err
		}

		ui.SetKey(k)
	}

	user := phonebook.User{
		Name:        name,
		Email:       email,
		Pubkey:      hex.EncodeToString(ui.Key().PublicKey),
		Description: description,
	}

	log.Debug().Msg("initializing client with created key")
	bcdb, err = client.NewClient(bcdbAddr, ui.Key())
	if err != nil {
		return err
	}

	log.Debug().Msg("register user")
	id, err := bcdb.Phonebook.Create(user)
	if err != nil {
		return errors.Wrap(err, "failed to register user")
	}

	// Update UserData with created id
	ui.ThreebotID = uint64(id)

	// Saving new seed struct
	if err := ui.Save(c.String("output")); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}

	fmt.Printf("Your ID is: %d\n", id)
	fmt.Printf("Seed saved to: %s\n", output)
	return nil
}
