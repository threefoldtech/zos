package main

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/identity"
	"github.com/threefoldtech/zosv2/modules/provision"
	"github.com/urfave/cli"
)

func createNetwork(c *cli.Context) error {
	farmID := c.String("farm")
	if farmID == "" {
		return fmt.Errorf("farm ID must be specified")
	}

	network, err := db.CreateNetwork(farmID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create network")
		return err
	}
	fmt.Printf("network created: %s\n", network.NetID)

	n := provision.Network{
		NetwokID: string(network.NetID),
	}
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r := provision.Reservation{
		ID:   id.String(),
		Type: provision.NetworkReservation,
		Data: raw,
	}

	nodeIDs := c.StringSlice("node")
	for _, nodeID := range nodeIDs {
		if err := store.Reserve(r, identity.StrIdentifier(nodeID)); err != nil {
			return err
		}
		fmt.Printf("network reservation sent for node ID %s\n", nodeID)
	}

	return nil
}

func addMember(c *cli.Context) error {
	nwID := c.String("network")
	if nwID == "" {
		return fmt.Errorf("network ID must be specified")
	}

	network, err := db.GetNetwork(modules.NetID(nwID))
	if err != nil {
		log.Error().Err(err).Msgf("network %s does not exists", nwID)
		return err
	}

	n := provision.Network{
		NetwokID: string(network.NetID),
	}
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	r := provision.Reservation{
		ID:   id.String(),
		Type: provision.NetworkReservation,
		Data: raw,
	}

	nodeIDs := c.StringSlice("node")
	for _, nodeID := range nodeIDs {
		if err := store.Reserve(r, identity.StrIdentifier(nodeID)); err != nil {
			return err
		}
		fmt.Printf("network reservation sent for node ID %s\n", nodeID)
	}

	return nil
}
