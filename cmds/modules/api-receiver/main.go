package apireceiver

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
	"github.com/threefoldtech/zosbase/pkg/api"
	"github.com/threefoldtech/zosbase/pkg/handlers"
	"github.com/threefoldtech/zosbase/pkg/network/namespace"
	"github.com/threefoldtech/zosbase/pkg/stubs"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zbus"
)

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "api-receiver",
	Usage: "handles mycelium messages",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var broker string = cli.String("broker")
	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus broker")
	}

	a, err := api.NewAPI(client, broker, "full")
	if err != nil {
		return errors.Wrap(err, "failed to create api")
	}

	idStub := stubs.NewIdentityManagerStub(client)
	sk := ed25519.PrivateKey(idStub.PrivateKey(cli.Context))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return err
	}

	man := substrate.NewManager("chain")

	msg, err := messenger.NewMessenger(
		"",
		60,
		man,
		messenger.WithIdentity(id),
		// messenger.WithAutoUpdateTwin(true),
	)
	if err != nil {
		return fmt.Errorf("failed to create substrate manager: %w", err)
	}
	defer msg.Close()

	server := messenger.NewJSONRPCServer(msg)

	hdrs := handlers.NewRpcHandler(a)
	handlers.RegisterHandlers(server, hdrs)

	netns, err := namespace.GetByName("ndmz")
	if err != nil {
		return fmt.Errorf("failed to get network namespace %s: %w", "ndmz", err)
	}
	defer netns.Close()
	return netns.Do(func(_ ns.NetNS) error {
		return server.Start(context.Background())
	})
}
