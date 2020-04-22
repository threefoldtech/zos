package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/builders"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

func cmdGraphNetwork(c *cli.Context) error {
	var (
		schema = c.GlobalString("schema")
		err    error
	)

	network, err := builders.LoadNetwork(schema)
	if err != nil {
		return err
	}

	outfile, err := os.Create(schema + ".dot")
	if err != nil {
		return err
	}

	return network.NetworkGraph(outfile)
}

func cmdCreateNetwork(c *cli.Context) error {
	name := c.String("name")
	if name == "" {
		return fmt.Errorf("network name cannot be empty")
	}
	ipRange := c.String("cidr")
	if ipRange == "" {
		return fmt.Errorf("ip range cannot be empty")
	}

	ipnet, err := types.ParseIPNet(ipRange)
	if err != nil {
		errors.Wrap(err, "invalid ip range")
	}

	networkBuilder := builders.NewNetworkBuilder(name)
	networkBuilder.WithIPRange(schema.IPRange{IPNet: ipnet.IPNet}).WithNetworkResources([]workloads.NetworkNetResource{})

	return writeWorkload(c.GlobalString("schema"), networkBuilder.Build())
}

func cmdsAddNode(c *cli.Context) error {
	var (
		schema = c.GlobalString("schema")

		nodeID = c.String("node")
		subnet = c.String("subnet")
		port   = c.Uint("port")

		forceHidden = c.Bool("force-hidden")
	)

	network, err := builders.LoadNetwork(schema)
	if err != nil {
		return err
	}

	return network.AddNode(schema, nodeID, subnet, port, forceHidden)
}

func cmdsAddAccess(c *cli.Context) error {
	var (
		schema = c.GlobalString("schema")

		nodeID   = c.String("node")
		subnet   = c.String("subnet")
		wgPubKey = c.String("wgpubkey")

		ip4 = c.Bool("ip4")
	)

	network, err := builders.LoadNetwork(schema)
	if err != nil {
		return err
	}

	return network.AddAccess(schema, nodeID, subnet, wgPubKey, ip4)
}

func cmdsRemoveNode(c *cli.Context) error {
	var (
		schema = c.GlobalString("schema")
		nodeID = c.String("node")
	)

	network, err := builders.LoadNetwork(schema)
	if err != nil {
		return err
	}

	return network.RemoveNode(schema, nodeID)
}
