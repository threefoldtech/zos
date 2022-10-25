package node

// import (
// 	"context"
// 	"fmt"
// 	"os/exec"
// 	"time"

// 	"github.com/pkg/errors"
// 	"github.com/rs/zerolog/log"
// 	"github.com/threefoldtech/substrate-client"
// 	"github.com/threefoldtech/zbus"
// 	"github.com/threefoldtech/zos/pkg"
// 	"github.com/threefoldtech/zos/pkg/network/bridge"
// 	"github.com/threefoldtech/zos/pkg/stubs"
// 	"github.com/threefoldtech/zos/pkg/zinit"
// )

// const (
// 	wolInterface = "zos"
// )

// var (
// 	errConnectionError = fmt.Errorf("connection error")
// )

// func enableWol(inf string) error {
// 	br, err := bridge.Get(inf)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to get zos bridge")
// 	}

// 	nics, err := bridge.ListNics(br, true)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to list attached nics to zos bridge")
// 	}

// 	for _, nic := range nics {
// 		if err := exec.Command("ethtools", "-s", nic.Attrs().Name, "wol", "g").Run(); err != nil {
// 			log.Error().Err(err).Str("nic", nic.Attrs().Name).Msg("failed to enable WOL for nic")
// 		}
// 	}

// 	return nil
// }

// type Manager struct {
// 	events *stubs.EventsStub
// 	sub    substrate.Manager
// 	farm   pkg.FarmID
// 	node   uint32
// 	ut     *Uptime
// }

// // TODO: enable wake up on lan `ethtool -s enp59s0 wol g`

// func NewPowerManager(cl zbus.Client, sub substrate.Manager, ut *Uptime, farm pkg.FarmID, node uint32) (*Manager, error) {
// 	events := stubs.NewEventsStub(cl)
// 	if err := enableWol(wolInterface); err != nil {
// 		return nil, err
// 	}

// 	mgr := &Manager{
// 		events: events,
// 		sub:    sub,
// 		farm:   farm,
// 		node:   node,
// 		ut:     ut,
// 	}

// 	return mgr, nil
// }

// func (m *Manager) getNode(nodeID uint32) (*substrate.Node, error) {
// 	client, err := m.sub.Substrate()
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to get connection to substrate")
// 	}
// 	defer client.Close()
// 	node, err := client.GetNode(nodeID)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "failed to get node information")
// 	}

// 	return node, nil
// }

// func (m *Manager) sync() error {
// 	node, err := m.getNode(m.node)
// 	if err != nil {
// 		return err
// 	}

// 	if !node.Power().IsDown {
// 		return nil
// 	}

// 	return m.shutdown()
// }

// func (m *Manager) powerUp(nodeID uint32) error {
// 	log.Info().Uint32("node", nodeID).Msg("powering on node")
// 	node, err := m.getNode(nodeID)
// 	if err != nil {
// 		return err
// 	}

// 	mac := ""
// 	for _, inf := range node.Interfaces {
// 		if inf.Name == "zos" {
// 			mac = inf.Mac
// 			break
// 		}
// 	}
// 	if mac == "" {
// 		return fmt.Errorf("can't find mac address of node '%d'", nodeID)
// 	}

// 	return exec.Command("ether-wake", "-i", "zos", mac).Run()

// }

// func (m *Manager) shutdown() error {
// 	log.Info().Msg("shutting down node because of chain")
// 	if _, err := m.ut.SendNow(); err != nil {
// 		log.Error().Err(err).Msg("failed to send uptime before shutting down")
// 	}

// 	// is down!
// 	init := zinit.Default()
// 	err := init.Shutdown()

// 	if errors.Is(err, zinit.ErrNotSupported) {
// 		log.Info().Msg("node does not support shutdown. rebooting to update")
// 		return init.Reboot()
// 	}

// 	return err
// }

// func (m *Manager) event(event *pkg.PowerChangeEvent) error {
// 	if event.FarmID != m.farm {
// 		return nil
// 	}

// 	log.Debug().
// 		Uint32("farm", uint32(m.farm)).
// 		Uint32("node", m.node).
// 		Msg("received power event for farm")
// 	if event.Kind == pkg.EventSubscribed {
// 		return m.sync()
// 	}

// 	if event.NodeID == m.node && event.Target.IsDown {
// 		log.Info().Msg("received an event to shutdown")
// 		return m.shutdown()
// 	}

// 	if event.NodeID != m.node && event.Target.IsUp {
// 		return m.powerUp(event.NodeID)
// 	}

// 	return nil
// }

// func (m *Manager) listen(ctx context.Context) error {
// 	log.Info().Msg("listening for power events")
// 	stream, err := m.events.PowerChangeEvent(ctx)
// 	if err != nil {
// 		return errors.Wrapf(errConnectionError, "failed to connect to zbus events: %s", err)
// 	}

// 	for event := range stream {
// 		if err := m.event(&event); err != nil {
// 			log.Error().Err(err).Msg("failed to process power event")
// 		}
// 	}

// 	return nil
// }

// // start processing time events.
// func (m *Manager) Start(ctx context.Context) {
// 	// first thing we need to make sure we are not suppose to be powered
// 	// off, so we need to sync with grid
// 	// 1) make sure at least one uptime was already sent
// 	m.ut.Mark.Done(ctx)
// 	if err := m.sync(); err != nil {
// 		log.Error().Err(err).Msg("failed to synchronize power status with grid")
// 	}

// 	// if the stream loop fails for any reason retry
// 	// unless context was cancelled
// 	for {
// 		err := m.listen(ctx)
// 		if err == nil {
// 			return
// 		}

// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-time.After(10 * time.Second):
// 		}
// 	}
// }
