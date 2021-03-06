package networkd

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/latency"
	"github.com/threefoldtech/zos/pkg/network/public"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/urfave/cli/v2"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/network/bootstrap"
	"github.com/threefoldtech/zos/pkg/network/ndmz"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/pkg/network/yggdrasil"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"
)

const redisSocket = "unix:///var/run/redis.sock"
const module = "network"

// Module is entry point for module
var Module cli.Command = cli.Command{
	Name:  "networkd",
	Usage: "handles network resources and user networks",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/networkd",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
	},
	Action: action,
}

func action(cli *cli.Context) error {
	var (
		root   string = cli.String("root")
		broker string = cli.String("broker")
	)

	waitYggdrasilBin()

	if err := bootstrap.DefaultBridgeValid(); err != nil {
		return errors.Wrap(err, "invalid setup")
	}

	client, err := zbus.NewRedisClient(broker)
	if err != nil {
		return errors.Wrap(err, "failed to connect to zbus broker")
	}

	directory, err := explorerClient()
	if err != nil {
		return errors.Wrap(err, "failed to connect to BCDB")
	}

	identity := stubs.NewIdentityManagerStub(client)
	nodeID := identity.NodeID()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	// choose exit interface for (br-pub)
	// - this is public_config.master if set
	// - otherwise we find the first nic with public ipv6 (that is not zos)
	// - finally we use zos if that is the last option.
	pub, err := getExitInterface(directory, nodeID.Identity())
	log.Debug().Err(err).Msgf("public interface configred: %+v", pub)
	if err != nil && err != ErrNoPubInterface {
		return errors.Wrap(err, "failed to get node public_config")
	}

	master, err := public.EnsurePublicSetup(nodeID, pub)
	if err != nil {
		return errors.Wrap(err, "failed to setup public bridge")
	}

	dmz := ndmz.New(nodeID.Identity(), master)

	if err := dmz.Create(ctx); err != nil {
		return errors.Wrap(err, "failed to create ndmz")
	}

	if err := ensureHostFw(ctx); err != nil {
		return errors.Wrap(err, "failed to host firewall rules")
	}

	ygg, err := startYggdrasil(ctx, identity.PrivateKey(), dmz)
	if err != nil {
		return errors.Wrap(err, "fail to start yggdrasil")
	}

	gw, err := ygg.Gateway()
	if err != nil {
		return errors.Wrap(err, "fail read yggdrasil subnet")
	}

	if err := dmz.SetIP(gw); err != nil {
		return errors.Wrap(err, "fail to configure yggdrasil subnet gateway IP")
	}

	// send another detail of network interfaces now that ndmz is created
	ndmzIfaces, err := dmz.Interfaces()
	if err != nil {
		return errors.Wrap(err, "failed to read ndmz network interfaces")
	}
	// already sends all the interfaces detail we find
	// this won't contains the ndmz IP yet, but this is OK.
	ifaces, err := getLocalInterfaces()
	if err != nil {
		return errors.Wrap(err, "failed to read local network interfaces")
	}

	ifaces = append(ifaces, ndmzIfaces...)

	if err := publishIfaces(ifaces, nodeID, directory); err != nil {
		return errors.Wrap(err, "failed to publish ndmz network interfaces to BCDB")
	}

	// watch modification of the address on the nic so we can update the explorer
	// with eventual new values
	go startAddrWatch(ctx, dmz, nodeID, directory, ifaces)

	log.Info().Msg("start zbus server")
	if err := os.MkdirAll(root, 0750); err != nil {
		return errors.Wrap(err, "fail to create module root")
	}

	networker, err := network.NewNetworker(identity, directory, root, dmz, ygg)
	if err != nil {
		return errors.Wrap(err, "error creating network manager")
	}

	if err := startServer(ctx, broker, networker); err != nil {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}

func startServer(ctx context.Context, broker string, networker pkg.Networker) error {

	server, err := zbus.NewRedisServer(module, broker, 1)
	if err != nil {
		log.Error().Err(err).Msgf("fail to connect to message broker server")
	}

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, networker)

	log.Info().
		Str("broker", broker).
		Uint("worker nr", 1).
		Msg("starting networkd module")

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	return nil
}

func waitYggdrasilBin() {
	log.Info().Msg("wait for yggdrasil binary to be available")
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 //forever
	_ = backoff.RetryNotify(func() error {
		_, err := exec.LookPath("yggdrasil")
		return err
	}, bo, func(err error, d time.Duration) {
		log.Warn().Err(err).Msgf("yggdrasil binary not found, retying in %s", d.String())
	})
}

func fetchPeerList() yggdrasil.PeerList {
	// Try to fetch public peer
	// If we failed to do so, use the fallback hardcoded peer list
	var pl yggdrasil.PeerList

	// Do not retry more than 4 times
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 4)

	fetchPeerList := func() error {
		p, err := yggdrasil.FetchPeerList()
		if err != nil {
			log.Debug().Err(err).Msg("failed to fetch yggdrasil peers")
			return err
		}
		pl = p
		return nil
	}

	err := backoff.Retry(fetchPeerList, bo)
	if err != nil {
		log.Error().Err(err).Msg("failed to read yggdrasil public peer list online, using fallback")
		pl = yggdrasil.PeerListFallback
	}

	return pl
}

func startYggdrasil(ctx context.Context, privateKey ed25519.PrivateKey, dmz ndmz.DMZ) (*yggdrasil.Server, error) {
	pl := fetchPeerList()
	peersUp := pl.Ups()
	endpoints := make([]string, len(peersUp))
	for i, p := range peersUp {
		endpoints[i] = p.Endpoint
	}

	// filter out the possible yggdrasil public node
	var filter latency.IPFilter
	ipv4Only, err := dmz.IsIPv4Only()
	if err != nil {
		return nil, errors.Wrap(err, "failed to check ipv6 support for dmz")
	}

	if ipv4Only {
		// if we are a hidden node,only keep ipv4 public nodes
		filter = latency.IPV4Only
	} else {
		// if we are a dual stack node, filter out all the nodes from the same
		// segment so we do not just connect locally
		ips, err := dmz.GetIP(ndmz.FamilyV6)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get ndmz public ipv6")
		}

		for _, ip := range ips {
			if ip.IP.IsGlobalUnicast() {
				filter = latency.ExcludePrefix(ip.IP[:8])
				break
			}
		}
	}

	ls := latency.NewSorter(endpoints, 5, filter)
	results := ls.Run(ctx)
	if len(results) == 0 {
		return nil, fmt.Errorf("cannot find public yggdrasil peer to connect to")
	}

	// select the best 3 public peers
	peers := make([]string, 3)
	for i := 0; i < 3; i++ {
		if len(results) > i {
			peers[i] = results[i].Endpoint
			log.Info().Str("endpoint", results[i].Endpoint).Msg("yggdrasill public peer selected")
		}
	}

	z, err := zinit.New("")
	if err != nil {
		return nil, err
	}

	cfg := yggdrasil.GenerateConfig(privateKey)
	cfg.Peers = peers

	server := yggdrasil.NewServer(z, &cfg)

	go func() {
		select {
		case <-ctx.Done():
			if err := server.Stop(); err != nil {
				log.Error().Err(err).Msg("error while stopping yggdrasil")
			}
			if err := z.Close(); err != nil {
				log.Error().Err(err).Msg("error while closing zinit client")
			}
			log.Info().Msg("yggdrasil stopped")
		}
	}()

	if err := server.Start(); err != nil {
		return nil, err
	}

	return server, nil
}

func startAddrWatch(ctx context.Context, dmz ndmz.DMZ, nodeID pkg.Identifier, cl client.Directory, ifaces []types.IfaceInfo) {

	ifaceNames := make([]string, len(ifaces))
	for i, iface := range ifaces {
		ifaceNames[i] = iface.Name
	}
	log.Info().Msgf("watched interfaces %v", ifaceNames)

	f := func() error {
		wl := NewWatchedLinks(dmz, ifaceNames, nodeID, cl)
		if err := wl.Forever(ctx); err != nil {
			log.Error().Err(err).Msg("error in address watcher")
			return err
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Minute
	bo.MaxElapsedTime = 0 // retry forever
	backoff.Retry(f, bo)
}

// instantiate the proper client based on the running mode
func explorerClient() (client.Directory, error) {
	client, err := app.ExplorerClient()
	if err != nil {
		return nil, err
	}

	return client.Directory, nil
}
