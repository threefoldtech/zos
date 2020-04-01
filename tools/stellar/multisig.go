package main

import (
	"fmt"

	"github.com/threefoldtech/zos/tools/explorer/pkg/stellar"

	"github.com/urfave/cli"
)

func create(c *cli.Context) error {
	seed := c.String("seed")
	network := c.String("network")
	asset := c.String("asset")
	from := c.String("from")
	destination := c.String("destination")
	amount := c.String("amount")

	wallet, err := stellar.New(seed, network, asset, nil)
	if err != nil {
		return err
	}
	tx, err := wallet.CreateMultisigTransaction(from, destination, amount)
	if err != nil {
		return err
	}
	fmt.Printf("Transaction to be signed: %s\n", tx)
	return nil
}

func signAndSubmit(c *cli.Context) error {
	seed := c.String("seed")
	network := c.String("network")
	transaction := c.String("transaction")

	wallet, err := stellar.New(seed, network, "", nil)
	if err != nil {
		return err
	}
	tx, err := wallet.SignAndSubmitMultisigTransaction(transaction)
	if err != nil {
		fmt.Printf("Transaction needs another signature: %s\n", tx)
		fmt.Println()
		return err
	}
	return nil
}
