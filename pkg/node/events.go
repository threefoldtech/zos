package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/mw"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	wolInterface = "zos"
	powerPort    = 8039
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

func (m *PowerServer) sync() error {
	node, err := m.getNode(m.node)
	if err != nil {
		return err
	}

	power := node.Power()
	// if the state is not down. nothing to do.
	// state down is only set by a call to the /power endpoint
	// it need to be requested by a neighbor node
	if !power.State.IsDown {
		return nil
	}

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
	log.Info().Uint32("node", uint32(node.ID)).Msg("powering on node")

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

func (m *PowerServer) powerRequest(ip string, r *powerRequest) error {
	data, err := json.Marshal(r)
	if err != nil {
		return errors.Wrap(err, "failed to build power off payload")
	}

	u := fmt.Sprintf("http://%s:%d", ip, powerPort)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrap(err, "failed to build power off request")
	}
	req, err = mw.SignedRequest(m.node, m.sk, req)
	if err != nil {
		return errors.Wrap(err, "failed to sign power off request")
	}

	// we need to call the remote node. and ask it to power off
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to make request to node ip ''", ip)
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusAccepted || response.StatusCode == http.StatusForbidden {
		// node either accepted the request or rejected it as being a power leader. hence to us
		// nothing is wrong
		return nil
	} else {
		body, _ := ioutil.ReadAll(response.Body)
		return errors.Wrapf(err, "got response '%s': %s", response.Status, string(body))
	}
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

	if event.NodeID != m.node && event.Target.IsDown {
		log.Info().Uint32("target", event.NodeID).Msg("received an event to shutdown")
		return m.powerDown(node)
	}

	if event.NodeID != m.node && event.Target.IsUp {
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
	if err := m.sync(); err != nil {
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
