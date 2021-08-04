package yggdrasil

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/jbenet/go-base58"

	"github.com/yggdrasil-network/yggdrasil-go/src/config"
)

// List of port used by yggdrasil
const (
	YggListenTCP       = 9943
	YggListenTLS       = 9944
	YggListenLinkLocal = 9945

	YggIface = "ygg0"
)

// GenerateConfig creates a new yggdrasil configuration and generate the
// box and signing key from the ed25519 Private key of the node
// this creates a mapping between a yggdrasil identity and the TFGrid identity
func GenerateConfig(privateKey ed25519.PrivateKey) (cfg config.NodeConfig) {
	cfg.IfMTU = 65535
	if privateKey != nil {
		cfg.PrivateKey = hex.EncodeToString(privateKey)

		signingPublicKey := privateKey.Public().(ed25519.PublicKey)
		cfg.PublicKey = hex.EncodeToString(signingPublicKey)

		cfg.NodeInfo = map[string]interface{}{
			"name": base58.Encode(signingPublicKey)[:6],
		}
	}

	cfg.MulticastInterfaces = []config.MulticastInterfaceConfig{
		{
			Regex:  ".*",
			Listen: true,
			Beacon: true,
			Port:   0,
		},
	}

	cfg.IfName = YggIface

	cfg.Listen = []string{
		fmt.Sprintf("tcp://[::]:%d", YggListenTCP),
		fmt.Sprintf("tls://[::]:%d", YggListenTLS),
	}

	return
}
