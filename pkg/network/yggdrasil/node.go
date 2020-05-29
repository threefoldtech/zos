package yggdrasil

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/gologme/log"
	"github.com/jbenet/go-base58"
	"github.com/threefoldtech/zos/pkg/crypto"

	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/module"
	"github.com/yggdrasil-network/yggdrasil-go/src/multicast"
	"github.com/yggdrasil-network/yggdrasil-go/src/tuntap"
	ygg "github.com/yggdrasil-network/yggdrasil-go/src/yggdrasil"
)

type Node struct {
	config *config.NodeConfig

	state     *config.NodeState
	core      ygg.Core
	tuntap    module.Module // tuntap.TunAdapter
	multicast module.Module // multicast.Multicast

	log *log.Logger
}

func GenerateConfig(privateKey ed25519.PrivateKey) config.NodeConfig {
	cfg := config.GenerateConfig()

	if privateKey != nil {
		cfg.SigningPrivateKey = hex.EncodeToString(privateKey)

		signingPublicKey := privateKey.Public().(ed25519.PublicKey)
		cfg.SigningPublicKey = hex.EncodeToString(signingPublicKey)

		encryptionPrivateKey := crypto.PrivateKeyToCurve25519(privateKey)
		cfg.EncryptionPrivateKey = hex.EncodeToString(encryptionPrivateKey[:])

		encryptionPublicKey := crypto.PublicKeyToCurve25519(signingPublicKey)
		cfg.EncryptionPublicKey = hex.EncodeToString(encryptionPublicKey[:])

		cfg.NodeInfo = map[string]interface{}{
			"name": base58.Encode(signingPublicKey),
		}
	}
	cfg.MulticastInterfaces = []string{"zos"}

	cfg.IfName = "ygg0"
	cfg.TunnelRouting.Enable = true
	// cfg.SessionFirewall.Enable = false

	// cfg.Listen = []string{
	// 	"tcp://0.0.0.0:4434",
	// 	"tls://0.0.0.0:4444",
	// }

	cfg.Peers = []string{
		"tls://91.121.92.51:4444",
		"tls://2a02:1802:5e:1001:ec4:7aff:fe30:82f8:4444",
		"tls://185.69.166.120:4444",
	}

	return *cfg
}

func New(cfg config.NodeConfig) *Node {

	node := &Node{
		config:    &cfg,
		log:       log.New(os.Stdout, "yggdrasil", log.Flags()),
		multicast: &multicast.Multicast{},
		tuntap:    &tuntap.TunAdapter{},
	}

	levels := [...]string{"error", "warn", "info", "debug", "trace"}
	for _, l := range levels {
		node.log.EnableLevel(l)
	}

	return node
}

func (n *Node) Start() error {
	var err error

	n.state, err = n.core.Start(n.config, n.log)
	if err != nil {
		return err
	}

	// Start the multicast interface
	n.multicast.Init(&n.core, n.state, n.log, nil)
	n.log.Infof("start multicast")
	if err := n.multicast.Start(); err != nil {
		n.log.Errorln("An error occurred starting multicast:", err)
	}
	// Start the TUN/TAP interface

	listener, err := n.core.ConnListen()
	if err != nil {
		return fmt.Errorf("Unable to get Dialer: %w", err)
	}
	dialer, err := n.core.ConnDialer()
	if err != nil {
		return fmt.Errorf("Unable to get Listener: %w", err)
	}
	n.tuntap.Init(&n.core, n.state, n.log, tuntap.TunOptions{Listener: listener, Dialer: dialer})
	if err := n.tuntap.Start(); err != nil {
		return fmt.Errorf("An error occurred starting TUN/TAP: %w", err)
	}

	return nil
}

func (n *Node) Shutdown() {
	n.multicast.Stop()
	n.tuntap.Stop()
	n.core.Stop()
}

func (n *Node) UpdateConfig(cfg config.NodeConfig) {
	n.log.Infoln("Reloading configuration")
	n.core.UpdateConfig(&cfg)
	n.tuntap.UpdateConfig(&cfg)
	n.multicast.UpdateConfig(&cfg)
}
