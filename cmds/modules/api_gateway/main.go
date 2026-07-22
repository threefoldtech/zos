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
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos_base/pkg/environment"
	"github.com/threefoldtech/zos_base/pkg/stubs"
	substrategw "github.com/threefoldtech/zos_base/pkg/substrate_gateway"
	"github.com/threefoldtech/zos_base/pkg/utils"
	zosapi "github.com/threefoldtech/zos_base/pkg/zos_api"
	"github.com/threefoldtech/zos_sdk_go/rmb-sdk-go/peer"
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

// spawnPeer creates an rmb peer bound to a fresh child context derived from parent and
// returns that child's cancel func, so the caller can tear the peer down later (e.g. to
// rebuild it with new relay urls) without leaving open relay connections behind.
//
// It retries NewPeer indefinitely — the node can't serve rmb without a peer. It must NOT
// cancel peerCtx on a failed attempt: NewPeer only starts its background goroutines after
// all of its error paths, so a failed attempt leaves nothing to clean up, and cancelling
// here would poison peerCtx — every subsequent retry (and the eventual live peer) would be
// built on an already-cancelled context and would never receive.
func spawnPeer(parent context.Context, seed string, manager substrate.Manager, handler peer.Handler, relayURLs []string) context.CancelFunc {
	peerCtx, cancel := context.WithCancel(parent)
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	_ = backoff.Retry(func() error {
		if _, err := peer.NewPeer(
			peerCtx,
			seed,
			manager,
			handler,
			peer.WithKeyType(peer.KeyTypeEd25519),
			peer.WithRelay(relayURLs...),
			peer.WithInMemoryExpiration(6*60*60), // 6 hours
		); err != nil {
			return fmt.Errorf("failed to start a new rmb peer: %w", err)
		}
		return nil
	}, bo)
	return cancel
}

func action(cli *cli.Context) error {
	var (
		msgBrokerCon = cli.String("broker")
		workerNr     = cli.Uint("workers")
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
	env := environment.MustGet()
	subURLs := env.SubstrateURL
	relayURLs := env.RelaysURLs
	manager, err := environment.GetSubstrate()
	if err != nil {
		return fmt.Errorf("failed to create substrate manager: %w", err)
	}

	router := peer.NewRouter()
	gw, err := substrategw.NewSubstrateGateway(manager, id)
	if err != nil {
		return fmt.Errorf("failed to create api gateway: %w", err)
	}

	_ = server.Register(zbus.ObjectID{Name: "api-gateway", Version: "0.0.1"}, gw)

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

	api, err := zosapi.NewZosAPI(manager, redis, msgBrokerCon)
	if err != nil {
		return fmt.Errorf("failed to create zos api: %w", err)
	}
	api.SetupRoutes(router)

	pair, err := id.KeyPair()
	if err != nil {
		return err
	}
	seed := hex.EncodeToString(pair.Seed())

	// cancel tears the current peer down (closing its relay connections) so it can be
	// rebuilt when the relay/substrate configuration changes.
	cancel := spawnPeer(ctx, seed, manager, router.Serve, relayURLs)

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
			updatedRelayURLs := env.RelaysURLs

			// make sure urls are sorted for comparison
			slices.Sort(subURLs)
			slices.Sort(relayURLs)
			slices.Sort(updatedSubURLs)
			slices.Sort(updatedRelayURLs)

			// skip update if substrate and relay urls did not change
			if slices.Equal(subURLs, updatedSubURLs) && slices.Equal(relayURLs, updatedRelayURLs) {
				log.Debug().Msg("zos-config doesn't have updated config to update the node with")
				continue
			}

			log.Debug().Strs("relays_urls", updatedRelayURLs).Strs("substrate_urls", updatedSubURLs).Msg("detected new update in configuration")

			manager, err = environment.GetSubstrate()
			if err != nil {
				// skip update if can't get sub manager
				log.Debug().Err(err).Msg("failed to get substrate manager")
				continue
			}
			// tear the current peer down and rebuild it with the updated urls
			log.Debug().Msg("cancelling current peer context to create a new one with updated urls")
			cancel()

			cancel = spawnPeer(ctx, seed, manager, router.Serve, updatedRelayURLs)

			relayURLs = updatedRelayURLs
			subURLs = updatedSubURLs

			log.Debug().Strs("relays_urls", relayURLs).Strs("substrate_urls", subURLs).Msg("updated substrate and relay urls")
		}
	}
}
