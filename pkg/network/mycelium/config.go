package mycelium

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"

	"github.com/decred/base58"
	"github.com/pkg/errors"
)

type NodeConfig struct {
	KeyFile    string
	TunName    string
	Peers      []string
	PrivateKey ed25519.PrivateKey
	PublicKey  string
	NodeInfo   map[string]interface{}
}

func (n *NodeConfig) FindPeers(ctx context.Context, filter ...Filter) error {
	// fetching a peer list goes as this
	// - Always include the list of peers from
	peers, err := fetchZosMyList()
	if err != nil {
		return errors.Wrap(err, "failed to get zos public peer list")
	}

	peers, err = peers.Ups(filter...)
	if err != nil {
		return errors.Wrap(err, "failed to filter out peer list")
	}

	n.Peers = peers
	return nil
}

// GenerateConfig creates a new mycelium configuration
func GenerateConfig(privateKey ed25519.PrivateKey) (cfg NodeConfig) {
	cfg = NodeConfig{
		KeyFile: confPath,
		TunName: tunName,
	}

	if privateKey != nil {
		cfg.PrivateKey = privateKey

		signingPublicKey := privateKey.Public().(ed25519.PublicKey)
		cfg.PublicKey = hex.EncodeToString(signingPublicKey)

		cfg.NodeInfo = map[string]interface{}{
			"name": base58.Encode(signingPublicKey)[:6],
		}
	}
	return
}
