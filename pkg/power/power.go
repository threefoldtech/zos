package power

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/network/bridge"
	"github.com/threefoldtech/zos/pkg/zinit"
	"github.com/vishvananda/netlink"
)

type PowerServer struct {
	cl       zbus.Client
	consumer *events.RedisConsumer
	sub      substrate.Manager

	farm     pkg.FarmID
	node     uint32
	twin     uint32
	sk       ed25519.PrivateKey
	identity substrate.Identity
	ut       *Uptime
	listen   string
	direct   *Direct
}

func NewPowerServer(
	cl zbus.Client,
	sub substrate.Manager,
	consumer *events.RedisConsumer,
	farm pkg.FarmID,
	node uint32,
	twin uint32,
	sk ed25519.PrivateKey,
	ut *Uptime) (*PowerServer, error) {

	identity, err := substrate.NewIdentityFromEd25519Key(sk)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialized identity")
	}

	lan, err := NewDirect(DefaultWolBridge)
	if err != nil {
		return nil, err
	}

	return &PowerServer{
		cl:       cl,
		sub:      sub,
		consumer: consumer,
		listen:   fmt.Sprintf(":%d", PowerServerPort),
		farm:     farm,
		node:     node,
		twin:     twin,
		sk:       sk,
		identity: identity,
		ut:       ut,
		direct:   lan,
	}, nil
}

const (
	DefaultWolBridge = "zos"
	PowerServerPort  = 8039
)

var (
	errConnectionError = fmt.Errorf("connection error")
)

func EnsureWakeOnLan(ctx context.Context) (bool, error) {
	inf, err := bridge.Get(DefaultWolBridge)
	if err != nil {
		return false, errors.Wrap(err, "failed to get zos bridge")
	}

	nics, err := bridge.ListNics(inf, true)
	if err != nil {
		return false, errors.Wrap(err, "failed to list attached nics to zos bridge")
	}

	filtered := nics[:0]
	for _, nic := range nics {
		if nic.Type() == "device" {
			filtered = append(filtered, nic)
		}
	}

	if len(filtered) != 1 {
		return false, fmt.Errorf("zos bridge has multiple interfaces")
	}

	nic := filtered[0].Attrs().Name
	log.Info().Str("nic", nic).Msg("enabling wol on interface")
	support, err := ValueOfFlag(ctx, nic, SupportsWakeOn)

	if errors.Is(err, ErrFlagNotFound) {
		// no support for
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "failed to detect support for wake on lan")
	}

	if !strings.Contains(support, string(MagicPacket)) {
		// no magic packet support either
		return false, nil
	}

	return true, SetWol(ctx, nic, MagicPacket)
}

func (p *PowerServer) syncSelf() error {
	cl, err := p.sub.Substrate()
	if err != nil {
		return err
	}

	power, err := cl.GetPowerTarget(p.node)
	if err != nil {
		return err
	}

	// power target is the state the node has to be in
	// while the node state is the actual state set by the node.

	// if target is up, and the node state is up, we do nothing
	// if target is up, but th state is down, we set the state to up and return
	// if target is down, we make sure state is down, then shutdown

	if power.Target.IsUp {
		if power.State.IsDown {
			_, err = cl.SetNodePowerState(p.identity, true)
			return errors.Wrap(err, "failed to set state to up")
		}

		return nil
	}

	// now the target must be down.
	// we need to shutdown

	if power.State.IsUp {
		_, err = cl.SetNodePowerState(p.identity, false)
		return errors.Wrap(err, "failed to set state to up")
	}

	// otherwise node need to get back to sleep.
	if err := p.shutdown(); err != nil {
		return errors.Wrap(err, "failed to issue shutdown")
	}

	return nil
}

func (p *PowerServer) powerUp(node *substrate.Node, reason string) error {
	log.Info().Uint32("node", uint32(node.ID)).Str("reason", reason).Msg("powering on node")

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

func (p *PowerServer) shutdown() error {
	log.Info().Msg("shutting down node because of chain")
	if _, err := p.ut.SendNow(); err != nil {
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

func (p *PowerServer) event(event *pkg.PowerTargetChangeEvent) error {
	if event.FarmID != p.farm {
		return nil
	}

	log.Debug().
		Uint32("farm", uint32(p.farm)).
		Uint32("node", p.node).
		Msg("received power event for farm")

	cl, err := p.sub.Substrate()
	if err != nil {
		return err
	}
	node, err := cl.GetNode(event.NodeID)
	if err != nil {
		return err
	}

	if event.NodeID == p.node && event.Target.IsDown {
		// we need to shutdown!
		if _, err := cl.SetNodePowerState(p.identity, false); err != nil {
			return errors.Wrap(err, "failed to set node power state to down")
		}

		return p.shutdown()
	} else if event.Target.IsDown {
		return nil
	}

	if event.Target.IsUp {
		log.Info().Uint32("target", event.NodeID).Msg("received an event to power up")
		return p.powerUp(node, "target is up")
	}

	return nil
}

func (p *PowerServer) recv(ctx context.Context) error {
	log.Info().Msg("listening for power events")
	stream, err := p.consumer.PowerTargetChange(ctx)
	if err != nil {
		return errors.Wrapf(errConnectionError, "failed to connect to zbus events: %s", err)
	}

	for event := range stream {
		if err := p.event(&event); err != nil {
			log.Error().Err(err).Msg("failed to process power event")
		}
	}

	return nil
}

// start processing time events.
func (p *PowerServer) events(ctx context.Context) error {
	// first thing we need to make sure we are not suppose to be powered
	// off, so we need to sync with grid
	// 1) make sure at least one uptime was already sent
	p.ut.Mark.Done(ctx)
	// 2) do we need to power off
	if err := p.syncSelf(); err != nil {
		return errors.Wrap(err, "failed to synchronize power status")
	}

	// if the stream loop fails for any reason retry
	// unless context was cancelled
	for {
		err := p.recv(ctx)
		if err == nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}

func (p *PowerServer) Start(ctx context.Context) error {
	return p.events(ctx)
}

type Direct struct {
	idx int
}

func NewDirect(inf string) (*Direct, error) {
	ln, err := netlink.LinkByName(inf)
	if err != nil {
		return nil, err
	}

	return &Direct{idx: ln.Attrs().Index}, nil
}

func (d *Direct) IsDirect(ip string) (bool, error) {
	ipT := net.ParseIP(ip)
	if ipT == nil {
		return false, fmt.Errorf("invalid ip address")
	}

	routes, err := netlink.RouteGet(ipT)
	// errors are returned only if network is unreachable
	// so we can just assume this is not direct. no extra checks
	if err != nil {
		return false, nil
	}

	for _, r := range routes {
		if r.Gw == nil && r.LinkIndex == d.idx {
			return true, nil
		}
	}

	return false, nil
}
