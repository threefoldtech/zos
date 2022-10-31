package node

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/mw"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	downTarget = "down"
)

type Elections interface {
	IsLeader() bool
}

type powerRequest struct {
	Leader uint32 `json:"leader"`
	Node   uint32 `json:"node"`
	Target string `json:"target"`
}

type PowerServer struct {
	cl  zbus.Client
	sub substrate.Manager

	farm      pkg.FarmID
	node      uint32
	sk        ed25519.PrivateKey
	ut        *Uptime
	listen    string
	elections Elections
	http      http.Client
}

func NewPowerServer(
	cl zbus.Client,
	sub substrate.Manager,
	farm pkg.FarmID,
	node uint32,
	sk ed25519.PrivateKey,
	ut *Uptime) (*PowerServer, error) {

	if err := enableWol(wolInterface); err != nil {
		return nil, err
	}

	return &PowerServer{
		cl:        cl,
		sub:       sub,
		listen:    fmt.Sprintf(":%d", PowerServerPort),
		farm:      farm,
		node:      node,
		sk:        sk,
		ut:        ut,
		elections: NewElectionsManager(cl, sub, node, farm),
		http:      newClient(),
	}, nil
}

const (
	wolInterface    = "zos"
	PowerServerPort = 8039
)

var (
	errConnectionError = fmt.Errorf("connection error")
)

func enableWol(inf string) error {
	br, err := bridge.Get(inf)
	if err != nil {
		return errors.Wrap(err, "failed to get zos bridge")
	}

	nics, err := bridge.ListNics(br, true)
	if err != nil {
		return errors.Wrap(err, "failed to list attached nics to zos bridge")
	}

	for _, nic := range nics {
		if err := exec.Command("ethtools", "-s", nic.Attrs().Name, "wol", "g").Run(); err != nil {
			log.Error().Err(err).Str("nic", nic.Attrs().Name).Msg("failed to enable WOL for nic")
		}
	}

	return nil
}

func (m *PowerServer) getNode(nodeID uint32) (*substrate.Node, error) {
	client, err := m.sub.Substrate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get connection to substrate")
	}
	defer client.Close()
	node, err := client.GetNode(nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node information")
	}

	return node, nil
}

func (m *PowerServer) synchronize(ctx context.Context) {
	for {

		if err := m.syncDownNodes(); err != nil {
			log.Error().Err(err).Msg("failed to synchronize neighbors power target")
			select {
			case <-time.After(1 * time.Minute):
				continue
			case <-ctx.Done():
				return
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Hour):
		}
	}
}

func (m *PowerServer) syncDownNodes() error {
	// this is called on start of power server
	// to try to bring all neighbors to proper state
	sub, err := m.sub.Substrate()
	if err != nil {
		return err
	}
	nodeIDs, err := sub.GetNodesByFarmID(uint32(m.farm))
	if err != nil {
		return errors.Wrap(err, "failed to list farm nodes")
	}

	for _, nodeID := range nodeIDs {
		if err := m.syncDownNode(sub, nodeID); err != nil {
			log.Error().Err(err).Uint32("node", nodeID).Msg("failed to sync node power status")
		}
	}

	return nil
}

func (m *PowerServer) syncDownNode(sub *substrate.Substrate, id uint32) error {
	node, err := sub.GetNode(id)
	if err != nil {
		return errors.Wrapf(err, "failed to get node '%d' from chain", id)
	}
	if node.Power().Target.IsUp ||
		node.Power().State.IsDown {
		return nil
	}

	// node target is down but node state is up.
	// means we should try to put it down. if it accepted
	// this we can simply try to ask it to power off. if that
	// didn't work it means probably the node is not reachable
	return m.powerDown(node)
}

func (m *PowerServer) powerRequest(ip string, in *powerRequest) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(in); err != nil {
		return err
	}

	url, err := buildUrl(ip, PowerServerPort, "power")
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return err
	}

	req, err = mw.SignedRequest(m.node, m.sk, req)
	if err != nil {
		return err
	}

	// note that we don't really care if the node accepted or rejected
	// the request. The next call can end as follows:
	// - Node is not in the same lan. hence we get back a connection error. We would
	//   just move on with our lives.
	// - Node is reachable but it doesn't accept the call may be the node thinks it shouldn't
	//   be powered off, that is also fine.
	// - Node actually powers off, then also nothing to d.
	resp, err := m.http.Do(req)
	if err == nil {
		resp.Body.Close()
	}

	// so by reaching here, we can return a nil error
	return nil
}

func (m *PowerServer) syncSelf() error {
	node, err := m.getNode(m.node)
	if err != nil {
		return err
	}

	power := node.Power()

	// state is down but target is up. we need to fix the
	// target
	if power.Target.IsUp {
		// TODO!: set node state to UP and return
		return nil
	}

	// if state was already down we need to call shutdown
	// this can be duo to a wake up to send uptime request
	if power.State.IsDown {
		return m.shutdown()
	}

	// otherwise do nothing
	return nil
}

func (m *PowerServer) powerUp(node *substrate.Node) error {
	log.Info().Uint32("node", uint32(node.ID)).Msg("powering on node")

	mac := ""
	for _, inf := range node.Interfaces {
		if inf.Name == "zos" {
			mac = inf.Mac
			break
		}
	}
	if mac == "" {
		return fmt.Errorf("can't find mac address of node '%d'", node.ID)
	}

	return exec.Command("ether-wake", "-i", "zos", mac).Run()

}

func (m *PowerServer) powerDown(node *substrate.Node) error {
	log.Debug().Uint32("node", uint32(node.ID)).Msg("powering off node")

	var ips []string
	for _, inf := range node.Interfaces {
		if inf.Name == "zos" {
			ips = inf.IPs
			break
		}
	}

	req := powerRequest{
		Leader: m.node,
		Node:   uint32(node.ID),
		Target: downTarget,
	}

	for _, ip := range ips {
		if err := m.powerRequest(ip, &req); err != nil {
			log.Error().Err(err).Uint32("target", uint32(node.ID)).Msg("failed to send power down request")
		}
	}

	return nil
}

func (m *PowerServer) shutdown() error {
	log.Info().Msg("shutting down node because of chain")
	if _, err := m.ut.SendNow(); err != nil {
		log.Error().Err(err).Msg("failed to send uptime before shutting down")
	}

	// is down!
	init := zinit.Default()
	err := init.Shutdown()

	if errors.Is(err, zinit.ErrNotSupported) {
		log.Info().Msg("node does not support shutdown. rebooting to update")
		return init.Reboot()
	}

	return err
}

func (m *PowerServer) event(event *pkg.PowerChangeEvent) error {
	if event.FarmID != m.farm {
		return nil
	}

	log.Debug().
		Uint32("farm", uint32(m.farm)).
		Uint32("node", m.node).
		Msg("received power event for farm")

	node, err := m.getNode(event.NodeID)
	if err != nil {
		return err
	}

	if event.NodeID == m.node {
		return nil
	}

	if event.Target.IsDown {
		log.Info().Uint32("target", event.NodeID).Msg("received an event to power down")
		return m.powerDown(node)
	}

	if event.Target.IsUp {
		log.Info().Uint32("target", event.NodeID).Msg("received an event to power up")
		return m.powerUp(node)
	}

	return nil
}

func (m *PowerServer) recv(ctx context.Context) error {
	log.Info().Msg("listening for power events")
	events := stubs.NewEventsStub(m.cl)
	stream, err := events.PowerChangeEvent(ctx)
	if err != nil {
		return errors.Wrapf(errConnectionError, "failed to connect to zbus events: %s", err)
	}

	for event := range stream {
		if err := m.event(&event); err != nil {
			log.Error().Err(err).Msg("failed to process power event")
		}
	}

	return nil
}

// start processing time events.
func (m *PowerServer) events(ctx context.Context) {
	// first thing we need to make sure we are not suppose to be powered
	// off, so we need to sync with grid
	// 1) make sure at least one uptime was already sent
	m.ut.Mark.Done(ctx)
	// 2) do we need to power off
	if err := m.syncSelf(); err != nil {
		log.Error().Err(err).Msg("failed to synchronize power status with grid")
	}

	// if the stream loop fails for any reason retry
	// unless context was cancelled
	for {
		err := m.recv(ctx)
		if err == nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func (p *PowerServer) self(r *http.Request) (interface{}, mw.Response) {
	stub := stubs.NewNetworkerStub(p.cl)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cfg, err := stub.GetPublicConfig(ctx)
	if err != nil {
		return nil, mw.Error(err)
	}

	type Self struct {
		ID      uint32     `json:"id"`
		Farm    pkg.FarmID `json:"farm"`
		Address string     `json:"address"`
		Access  bool       `json:"access"`
	}

	return Self{
		ID:      p.node,
		Farm:    p.farm,
		Address: hex.EncodeToString(p.sk.Public().(ed25519.PublicKey)),
		Access:  !cfg.IsEmpty(),
	}, nil
}

func (p *PowerServer) power(r *http.Request) (interface{}, mw.Response) {
	var request powerRequest
	if p.elections.IsLeader() {
		// I am a leader, i don't listen to anyone
		return nil, mw.Forbidden(fmt.Errorf("is a leader node"))
	}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&request); err != nil {
		return nil, mw.BadRequest(err)
	}

	leader := mw.TwinID(r.Context())
	// the leader id in the request header doesn't match the body
	if leader != request.Leader {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid leader id in request"))
	}

	if request.Target != downTarget {
		return nil, mw.BadRequest(fmt.Errorf("unknown power target '%s'", request.Target))
	}

	sub, err := p.sub.Substrate()
	if err != nil {
		return nil, mw.Error(err)
	}

	defer sub.Close()
	leaderNode, err := sub.GetNode(leader)
	if err != nil {
		return nil, mw.Error(errors.Wrap(err, "failed to get leader node"))
	}

	if leaderNode.FarmID != types.U32(p.farm) {
		return nil, mw.UnAuthorized(fmt.Errorf("requesting node is not in the same farm"))
	}

	// TODO: set current state on chain before powering off.
	if err := p.shutdown(); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (p *PowerServer) Start(ctx context.Context) error {
	router := mux.NewRouter()
	signer := mw.NewSigner(p.sk)

	go p.events(ctx)
	go p.synchronize(ctx)

	// always sign responses
	router.Handle("/self", signer.Action(p.self)).Methods("GET")
	authorized := router.PathPrefix("/").Subrouter()
	twins, err := provision.NewSubstrateTwins(p.sub)
	if err != nil {
		return err
	}
	auth := mw.NewAuthMiddleware(twins)
	authorized.Use(auth.Middleware)
	authorized.Handle("/power", signer.Action(p.power)).Methods("POST")

	server := http.Server{
		Addr:    p.listen,
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()

	err = server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to start power manager handler")
	}

	return nil
}
