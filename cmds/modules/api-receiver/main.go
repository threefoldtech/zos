package apireceiver

import (
	"crypto/ed25519"
	"fmt"

	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/messenger"
	"github.com/threefoldtech/zosbase/pkg/api"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/handlers"
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

	man, err := environment.GetSubstrate()
	if err != nil {
		return fmt.Errorf("failed to get substrate manager: %w", err)
	}

	ctx := cli.Context
	msg, err := messenger.NewMessenger(
		"",
		60,
		man,
		messenger.WithIdentity(id),
		messenger.WithAutoUpdateTwin(false),
	)
	if err != nil {
		return fmt.Errorf("failed to create substrate manager: %w", err)
	}
	defer msg.Close()

	server := messenger.NewJSONRPCServer(msg)

	hdrs := handlers.NewRpcHandler(a)
	handlers.RegisterHandlers(server, hdrs)

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// block forever
	<-ctx.Done()
	return nil
}
