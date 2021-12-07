package yggdrasil

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/network/latency"
	"github.com/threefoldtech/zos/pkg/zinit"
)

func fetchPeerList() Peers {
	// Try to fetch public peer
	// If we failed to do so, use the fallback hardcoded peer list
	var pl Peers

	// Do not retry more than 4 times
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 4)

	fetchPeerList := func() error {
		p, err := FetchPeerList()
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
		pl = PeerListFallback
	}

	return pl
}

func EnsureYggdrasil(ctx context.Context, privateKey ed25519.PrivateKey, ns YggdrasilNamespace) (*YggServer, error) {
	pl := fetchPeerList()

	// Filter out all the nodes from the same
	// segment so we do not just connect locally
	ips, err := ns.GetIPs() // returns ipv6 only
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ndmz public ipv6")
	}

	var ranges Ranges
	for _, ip := range ips {
		if ip.IP.IsGlobalUnicast() {
			ranges = append(ranges, ip)
		}
	}

	filter := Exclude(ranges)

	peersUp, err := pl.Ups(filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get peers list")
	}

	endpoints := make([]string, len(peersUp))
	for i, p := range peersUp {
		endpoints[i] = p.Endpoint
	}

	ls := latency.NewSorter(endpoints, 5)
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

	z := zinit.Default()

	cfg := GenerateConfig(privateKey)
	cfg.Peers = peers

	server := NewYggServer(&cfg)
	if err := server.Ensure(z, ns.Name()); err != nil {
		return nil, err
	}

	gw, err := server.Gateway()
	if err != nil {
		return nil, errors.Wrap(err, "fail read yggdrasil subnet")
	}

	if err := ns.SetYggIP(gw, nil); err != nil {
		return nil, errors.Wrap(err, "fail to configure yggdrasil subnet gateway IP")
	}

	return server, nil
}
