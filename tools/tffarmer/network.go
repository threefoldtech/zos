package main

import (
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func giveAlloc(c *cli.Context) error {

	farmID, err := loadFarmID(c.String("seed"))
	if err != nil {
		log.Error().Err(err).Msg("impossible to load farm id, user register command first")
		return err
	}

	alloc := c.Args().First()
	_, allocation, err := net.ParseCIDR(alloc)
	if err != nil {
		log.Error().Err(err).Msg("prefix format not valid, use ip/mask")
		return err
	}

	if err := db.RegisterAllocation(farmID, allocation); err != nil {
		log.Error().Err(err).Msg("failed to register prefix")
		return err
	}

	fmt.Println("prefix registered successfully")
	return nil
}

func getAlloc(c *cli.Context) error {

	farm := c.Args().First()
	alloc, err := db.RequestAllocation(strID(farm))
	if err != nil {
		log.Error().Err(err).Msg("failed to get an allocation")
		return err
	}

	fmt.Printf("allocation received: %s\n", alloc.String())
	return nil
}

func configPublic(c *cli.Context) error {
	ip, ipnet, err := net.ParseCIDR(c.String("ip"))
	if err != nil {
		return err
	}
	ipnet.IP = ip
	gw := net.ParseIP(c.String("gw"))
	iface := c.String("iface")
	node := c.Args().First()

	if err := db.ConfigurePublicIface(strID(node), ipnet, gw, iface); err != nil {
		return err
	}
	fmt.Printf("public interface configured on node %s\n", node)
	return nil
}

func selectExit(c *cli.Context) error {
	node := c.Args().First()

	if err := db.SelectExitNode(strID(node)); err != nil {
		return err
	}
	fmt.Printf("Node %s marked as exit node\n", node)
	return nil
}
