package apigateway

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg/environment"
	"github.com/threefoldtech/zosbase/pkg/stubs"
	substrategw "github.com/threefoldtech/zosbase/pkg/substrate_gateway"
	"github.com/threefoldtech/zosbase/pkg/utils"
	zosapi "github.com/threefoldtech/zosbase/pkg/zos_api"
	"github.com/urfave/cli/v2"
)

const module = "api-gateway"

// Module entry point
var Module cli.Command = cli.Command{
	Name:  module,
	Usage: "handles outgoing chain calls and incoming rmb calls",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.UintFlag{
			Name:  "workers",
			Usage: "number of workers `N`",
			Value: 1,
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon string = cli.String("broker")
		workerNr     uint   = cli.Uint("workers")
	)

	server, err := zbus.NewRedisServer(module, msgBrokerCon, workerNr)
	if err != nil {
		return fmt.Errorf("fail to connect to message broker server: %w", err)
	}
	redis, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return fmt.Errorf("fail to connect to message broker server: %w", err)
	}
	idStub := stubs.NewIdentityManagerStub(redis)

	sk := ed25519.PrivateKey(idStub.PrivateKey(cli.Context))
	id, err := substrate.NewIdentityFromEd25519Key(sk)
	log.Info().Str("address", id.Address()).Msg("node address")
	if err != nil {
		return err
	}

	subURLs := environment.MustGet().SubstrateURL
	relayURLs := environment.GetRelaysURLs()
	manager, err := environment.GetSubstrate()
	if err != nil {
		return fmt.Errorf("failed to create substrate manager: %w", err)
	}

	router := peer.NewRouter()
	gw, err := substrategw.NewSubstrateGateway(manager, id)
	if err != nil {
		return fmt.Errorf("failed to create api gateway: %w", err)
	}

	server.Register(zbus.ObjectID{Name: "api-gateway", Version: "0.0.1"}, gw)

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	go func() {
		for {
			if err := server.Run(ctx); err != nil && err != context.Canceled {
				log.Error().Err(err).Msg("unexpected error")
				continue
			}

			break
		}
	}()

	// no need to restart zos-api here as it only tries to get farm and twin, donesn't mentain any open connections
	api, err := zosapi.NewZosAPI(manager, redis, msgBrokerCon)
	if err != nil {
		return fmt.Errorf("failed to create zos api: %w", err)
	}
	api.SetupRoutes(router)

	pair, err := id.KeyPair()
	if err != nil {
		return err
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0

	peerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	backoff.Retry(func() error {
		_, err = peer.NewPeer(
			peerCtx,
			hex.EncodeToString(pair.Seed()),
			manager,
			router.Serve,
			peer.WithKeyType(peer.KeyTypeEd25519),
			peer.WithRelay(relayURLs...),
			peer.WithInMemoryExpiration(6*60*60), // 6 hours
		)
		if err != nil {
			return fmt.Errorf("failed to start a new rmb peer: %w", err)
		}

		return nil
	}, bo)

	log.Info().
		Str("broker", msgBrokerCon).
		Uint("worker nr", workerNr).
		Msg("starting api-gateway module")

	// block forever
	for {
		select {
		case <-ctx.Done():
			return nil
			// check if we need to run an update on the peer and only do the update if all the changes are done successfully
		case <-time.After(10 * time.Minute):
			env, err := environment.Get()
			if err != nil {
				// skip update if we can't get env
				log.Debug().Err(err).Msg("failed to load node environment")
				continue
			}

			updatedSubURLs := env.SubstrateURL
			updatedRelayURLs := environment.GetRelaysURLs()

			// make sure urls are sorted for comparison
			slices.Sort(subURLs)
			slices.Sort(relayURLs)
			slices.Sort(updatedSubURLs)
			slices.Sort(updatedRelayURLs)

			// skip update if the urls did not change
			if slices.Equal(subURLs, updatedSubURLs) && slices.Equal(relayURLs, updatedRelayURLs) {
				continue
			}

			log.Debug().Strs("relays_urls", updatedRelayURLs).Strs("substrate_urls", env.SubstrateURL).Msg("detected new update in configuration")

			manager, err = environment.GetSubstrate()
			if err != nil {
				// skip update if can't get sub manager
				log.Debug().Err(err).Msg("failed to get substrate manager")
				continue
			}

			newPeerCtx, newCancel := context.WithCancel(ctx)
			if _, err = peer.NewPeer(
				newPeerCtx,
				hex.EncodeToString(pair.Seed()),
				manager,
				router.Serve,
				peer.WithKeyType(peer.KeyTypeEd25519),
				peer.WithRelay(updatedRelayURLs...),
				peer.WithInMemoryExpiration(6*60*60), // 6 hours
			); err != nil {
				newCancel()
				log.Debug().Err(err).Msg("failed to start a new rmb peer")
				continue
			}

			if !slices.Equal(subURLs, updatedSubURLs) {
				err = gw.UpdateSubstrateGatewayConnection(manager)
				if err != nil {
					newCancel()
					log.Debug().Err(err).Msg("failed to update substrate gateway with new manager")
					continue
				}
			}

			cancel()
			peerCtx = newPeerCtx
			cancel = newCancel

			relayURLs = updatedRelayURLs
			subURLs = updatedSubURLs

			log.Debug().Strs("relays_urls", relayURLs).Strs("substrate_urls", subURLs).Msg("updated substrate and relay urls")
		}
	}
}
