package node

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"
)

var (
	errConnectionError = fmt.Errorf("connection error")
)

type Manager struct {
	events *stubs.EventsStub
	sub    substrate.Manager
	farm   pkg.FarmID
	node   uint32
	ut     *Uptime
}

func NewManager(cl zbus.Client, sub substrate.Manager, ut *Uptime, farm pkg.FarmID, node uint32) *Manager {
	events := stubs.NewEventsStub(cl)
	mgr := &Manager{
		events: events,
		sub:    sub,
		farm:   farm,
		node:   node,
		ut:     ut,
	}

	return mgr
}

func (m *Manager) getNode(nodeID uint32) (*substrate.Node, error) {
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

func (m *Manager) sync() error {
	node, err := m.getNode(m.node)
	if err != nil {
		return err
	}

	if !node.Power().IsDown {
		return nil
	}

	return m.shutdown()
}

func (m *Manager) powerUp(nodeID uint32) error {
	log.Info().Uint32("node", nodeID).Msg("powering on node")
	node, err := m.getNode(nodeID)
	if err != nil {
		return err
	}

	mac := ""
	for _, inf := range node.Interfaces {
		if inf.Name == "zos" {
			mac = inf.Mac
			break
		}
	}
	if mac == "" {
		return fmt.Errorf("can't find mac address of node '%d'", nodeID)
	}

	return exec.Command("ether-wake", "-i", "zos", mac).Run()

}

func (m *Manager) shutdown() error {
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

func (m *Manager) event(event *pkg.PowerChangeEvent) error {
	if event.FarmID != m.farm {
		return nil
	}

	log.Debug().
		Uint32("farm", uint32(m.farm)).
		Uint32("node", m.node).
		Msg("received power event for farm")
	if event.Kind == pkg.EventSubscribed {
		return m.sync()
	}

	if event.NodeID == m.node && event.Target.IsDown {
		log.Info().Msg("received an event to shutdown")
		return m.shutdown()
	}

	if event.NodeID != m.node && event.Target.IsUp {
		return m.powerUp(event.NodeID)
	}

	return nil
}

func (m *Manager) listen(ctx context.Context) error {
	log.Info().Msg("listening for power events")
	stream, err := m.events.PowerChangeEvent(ctx)
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
func (m *Manager) Start(ctx context.Context) {
	// if the stream loop fails for any reason retry
	// unless context was cancelled
	for {
		err := m.listen(ctx)
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
