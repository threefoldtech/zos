package node

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/mw"
	"github.com/threefoldtech/zos/pkg/network/public"
)

const (
	retryElectionTime        = 30 * time.Minute
	retryElectionOnErrorTime = 30 * time.Second
	nodeResponseTimeout      = 5 * time.Second
)

type electionsManager struct {
	sub    substrate.Manager
	farmID pkg.FarmID
	nodeID uint32
	leader atomic.Bool
}

func NewElectionsManager(sub substrate.Manager, nodeID uint32, farmID pkg.FarmID) Elections {
	leader := atomic.Bool{}
	leader.Store(true)
	return &electionsManager{sub: sub, nodeID: nodeID, farmID: farmID, leader: leader}
}

func (e *electionsManager) IsLeader() bool {
	return e.leader.Load()
}

func (e *electionsManager) Start(ctx context.Context) {
	for {
		err := e.elect()
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

func (e *electionsManager) elect() error {
	// set leader to true if node has public config
	if public.HasPublicSetup() {
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
		node, err := sub.GetNode(nodeID)
		if err != nil {
			return errors.Wrapf(err, "failed to load node: %d", nodeID)
		}
		twin, err := sub.GetTwin(uint32(node.TwinID))
		if err != nil {
			return errors.Wrapf(err, "failed to load twin of node: %d", nodeID)
		}
		if checkSameLAN(node, twin.Account.PublicKey()) {
			nodes = append(nodes, nodeID)
		}

	}

	// get smallest nodeID and set leader to true if the node has the smallest nodeID
	var smallestID uint32 = 1000000000
	for _, nodeID := range nodes {
		if nodeID < smallestID {
			smallestID = nodeID
		}
	}
	if e.nodeID < smallestID {
		e.leader.Store(true)
		return nil
	}

	// set leader to false otherwise
	e.leader.Store(false)
	return nil
}

func checkSameLAN(node *substrate.Node, publicKey ed25519.PublicKey) bool {
	for _, i := range node.Interfaces {
		if i.Name != "zos" {
			continue
		}
		for _, ip := range i.IPs {
			if verifyNodeResponse(ip, publicKey) {
				return true
			}
		}
	}
	return false
}

func verifyNodeResponse(ip string, publicKey ed25519.PublicKey) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		log.Error().Msgf("failed to parse IP: %s", parsedIP)
		return false
	}
	var response *http.Response
	var err error
	client := http.Client{
		Timeout: nodeResponseTimeout,
	}
	if parsedIP.To4() != nil {
		response, err = client.Get(fmt.Sprintf("http://%s:%d/self", ip, powerPort))

	} else {
		response, err = client.Get(fmt.Sprintf("http://[%s]:%d/self", ip, powerPort))
	}
	if err != nil {
		return false
	}
	_, err = mw.VerifyResponse(publicKey, response)

	return err == nil
}
