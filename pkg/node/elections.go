package node

import (
	"context"
	"fmt"
	"net/http"
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
)

type electionsManager struct {
	sub    substrate.Manager
	farmID pkg.FarmID
	nodeID uint32
	leader bool
}

func NewElectionsManager(sub substrate.Manager, nodeID uint32, farmID pkg.FarmID) Elections {
	return &electionsManager{sub: sub, nodeID: nodeID, farmID: farmID, leader: true}
}

func (e *electionsManager) IsLeader() bool {
	return e.leader
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
		e.leader = true
		return nil
	}

	sub, err := e.sub.Substrate()
	if err != nil {
		return errors.Wrap(err, "failed to get connection to substrate")
	}
	defer sub.Close()
	farmNodes, err := sub.GetNodeByFarmID(uint32(e.farmID))
	if err != nil {
		return errors.Wrapf(err, "failed to get nodes on farm: %d", e.farmID)
	}

	// get nodes on the same LAN
	var nodes []uint32
	for _, nodeID := range farmNodes {
		node, err := sub.GetNode(nodeID)
		if err != nil {
			return errors.Wrapf(err, "failed to load node: %d", nodeID)
		}
		twin, err := sub.GetTwin(uint32(node.TwinID))
		if err != nil {
			return errors.Wrapf(err, "failed to load twin of node: %d", nodeID)
		}
		// loop over node IPs
		for _, i := range node.Interfaces {
			if i.Name != "zos" {
				continue
			}
			for _, ip := range i.IPs {
				response, err := http.Get("http://" + ip + ":" + fmt.Sprint(powerPort) + "/self")
				if err != nil {
					continue
				}
				_, err = mw.VerifyResponse(twin.Account.PublicKey(), response)
				if err == mw.ErrInvalidSignature {
					continue
				}
				if err != nil {
					return errors.Wrapf(err, "error verifying response from node: %d, ip: %s", nodeID, ip)
				}
				nodes = append(nodes, nodeID)
				break
			}
		}

	}

	// get smallest nodeID and set leader to true if the node has the smallest nodeID
	var smallestID uint32
	for _, nodeID := range nodes {
		if nodeID < smallestID {
			smallestID = nodeID
		}
	}
	if e.nodeID == smallestID {
		e.leader = true
		return nil
	}

	// set leader to false otherwise
	e.leader = false
	return nil
}
