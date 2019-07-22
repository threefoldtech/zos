package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/urfave/cli"
)

func cmdCreateNetwork(c *cli.Context) error {
	network, err := createNetwork(c.String("farm"))
	if err != nil {
		return err
	}

	return output(c.GlobalString("output"), network)
}

func cmdsAddNode(c *cli.Context) error {
	var (
		network = &modules.Network{}
		input   = c.GlobalString("input")
		netID   = c.String("network")
		err     error
	)

	network, err = loadNetwork(input, netID)
	if err != nil {
		return err
	}

	for _, nodeID := range c.StringSlice("node") {
		network, err = addNode(network, nodeID, "", 0)
		if err != nil {
			return errors.Wrap(err, "failed to add the node into the network object")
		}
	}

	return output(c.GlobalString("output"), network)
}
func cmdsAddUser(c *cli.Context) error {
	var (
		network = &modules.Network{}
		input   = c.GlobalString("input")
		netID   = c.String("network")
		err     error
	)

	network, err = loadNetwork(input, netID)
	if err != nil {
		return err
	}

	network, err = addUser(network, c.String("user"))
	if err != nil {
		return errors.Wrap(err, "failed to add the node into the network object")
	}

	return output(c.GlobalString("output"), network)
}

func cmdsReserveNetwork(c *cli.Context) error {
	var (
		network = &modules.Network{}
		input   = c.GlobalString("input")
		err     error
	)

	network, err = loadNetwork(input, "")
	if err != nil {
		return err
	}

	return reserveNetwork(network)
}

func loadNetwork(name, netID string) (network *modules.Network, err error) {
	network = &modules.Network{}
	if name != "" {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(network); err != nil {
			return nil, errors.Wrapf(err, "failed to decode json encoded network at %s", name)
		}
		return network, nil
	}
	return db.GetNetwork(modules.NetID(netID))
}

func output(name string, network *modules.Network) (err error) {
	var w io.Writer = os.Stdout
	if name != "" {
		w, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
		if err != nil {
			return err
		}
	}

	return json.NewEncoder(w).Encode(network)
}
