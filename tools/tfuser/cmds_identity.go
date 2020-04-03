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

	eud, err := identity.LoadUserIdentity(output)
	if err == nil {
		fmt.Printf("A seed already exists at: %s\n", output)
		fmt.Printf("Identity: %s\n", eud.Key.Identity())
		return nil
	}

	ud := identity.UserData{}

	if mnemonic != "" {
		log.Info().Msg("building key using existing mnemonic")
		ui := &identity.UserIdentity{
			Mnemonic:   mnemonic,
			ThreebotID: 0,
		}

		ud, err = identity.LoadUserIdentityObject(ui)
		if err != nil {
			return err
		}

	} else {
		k, err := identity.GenerateKeyPair()
		if err != nil {
			return err
		}

		ud.Key = k
	}

	user := phonebook.User{
		Name:        name,
		Email:       email,
		Pubkey:      hex.EncodeToString(ud.Key.PublicKey),
		Description: description,
	}

	log.Debug().Msg("initializing client with created key")
	bcdb, err = client.NewClient(bcdbAddr, ud.Key)
	if err != nil {
		return err
	}

	log.Debug().Msg("register user")
	id, err := bcdb.Phonebook.Create(user)
	if err != nil {
		return errors.Wrap(err, "failed to register user")
	}

	// Update UserData with created id
	ud.ThreebotID = uint64(id)

	// Saving new seed struct
	if err := identity.SaveUserData(&ud, c.String("output")); err != nil {
		return errors.Wrap(err, "failed to save seed")
	}

	fmt.Printf("Your ID is: %d\n", id)
	fmt.Printf("Seed saved to: %s\n", output)
	return nil
}
