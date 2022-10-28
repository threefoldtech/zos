package node

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/mw"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	retryElectionTime        = 30 * time.Minute
	retryElectionOnErrorTime = 30 * time.Second
	nodeResponseTimeout      = 5 * time.Second
)

type electionsManager struct {
	zbus   zbus.Client
	sub    substrate.Manager
	farmID pkg.FarmID
	nodeID uint32
	client http.Client
	leader atomic.Bool
}

func NewElectionsManager(cl zbus.Client, sub substrate.Manager, nodeID uint32, farmID pkg.FarmID) Elections {
	var leader atomic.Bool
	leader.Store(true)

	return &electionsManager{
		zbus:   cl,
		sub:    sub,
		nodeID: nodeID,
		farmID: farmID,
		client: newClient(),
		leader: leader,
	}
}

func (e *electionsManager) IsLeader() bool {
	return e.leader.Load()
}

func (e *electionsManager) Start(ctx context.Context) {
	for {
		err := e.elect(ctx)
		if err != nil {
			log.Error().Err(err).Msg("elections failed")
			select {
			case <-time.After(retryElectionOnErrorTime):
				continue
			case <-ctx.Done():
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(retryElectionTime):
			continue
		}
	}
}

func (e *electionsManager) elect(ctx context.Context) error {
	// set leader to true if node has public config
	stub := stubs.NewNetworkerStub(e.zbus)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg, err := stub.GetPublicConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check for public config")
	}

	if !cfg.IsEmpty() {
		e.leader.Store(true)
		return nil
	}

	sub, err := e.sub.Substrate()
	if err != nil {
		return errors.Wrap(err, "failed to get connection to substrate")
	}
	defer sub.Close()
	farmNodes, err := sub.GetNodesByFarmID(uint32(e.farmID))
	if err != nil {
		return errors.Wrapf(err, "failed to get nodes on farm: %d", e.farmID)
	}

	// get nodes on the same LAN
	var nodes []uint32
	for _, nodeID := range farmNodes {
		if nodeID == e.nodeID {
			continue
		}
		reachable, err := e.checkNode(sub, nodeID)
		if err != nil {
			log.Error().Uint32("node", nodeID).Err(err).Msg("failed to check node reachability")
			continue
		}
		if reachable {
			nodes = append(nodes, nodeID)
		}
	}

	// if this node id is greater than any of the ids in the list
	// then you can't be the leader, set it to false and return
	for _, nodeID := range nodes {
		if e.nodeID > nodeID {
			e.leader.Store(false)
			return nil
		}
	}
	// otherwise you can be the leader
	e.leader.Store(true)
	return nil
}

func (e *electionsManager) checkNode(sub *substrate.Substrate, nodeID uint32) (bool, error) {
	node, err := sub.GetNode(nodeID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to load node: %d", nodeID)
	}

	twin, err := sub.GetTwin(uint32(node.TwinID))
	if err != nil {
		return false, errors.Wrapf(err, "failed to load twin of node: %d", nodeID)
	}

	return e.checkSameLAN(node, twin.Account.PublicKey()), nil
}

func (e *electionsManager) checkSameLAN(node *substrate.Node, publicKey ed25519.PublicKey) bool {
	for _, i := range node.Interfaces {
		if i.Name != "zos" {
			continue
		}
		for _, ip := range i.IPs {
			if err := e.verifyNodeResponse(ip, publicKey); err != nil {
				log.Debug().Err(err).Str("ip", ip).Msg("ip not reachable")
			}
			return true
		}
	}
	return false
}

func (e *electionsManager) verifyNodeResponse(ip string, publicKey ed25519.PublicKey) error {
	url, err := buildUrl(ip, PowerServerPort, "self")
	if err != nil {
		return err
	}

	response, err := e.client.Get(url)
	if err != nil {
		return err
	}
	_, err = mw.VerifyResponse(publicKey, response)
	return err
}

// newClient creates a new http client with a custom TCP timeout. We need to timeout only
// on establishing the connection (the underlying tcp connection) but once connection is established
// it's okay to have the connection open for longer to use it for requests.
func newClient() http.Client {
	dialer := &net.Dialer{
		Timeout:   nodeResponseTimeout,
		KeepAlive: 10 * time.Second,
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return http.Client{
		Transport: transport,
	}
}

// buildUrl builds correct url given ip
func buildUrl(ip string, port uint16, path ...string) (string, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid ip address '%s'", ip)
	}

	p := filepath.Join(path...)

	if parsedIP.To4() != nil {
		return fmt.Sprintf("http://%s:%d/%s", ip, port, p), nil
	} else {
		return fmt.Sprintf("http://[%s]:%d/%s", ip, port, p), nil
	}
}
