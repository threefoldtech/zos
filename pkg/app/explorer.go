package app

import (
	"crypto/ed25519"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/identity"
)

const seedPath = "/var/cache/modules/identityd/seed.txt"

// ExplorerClient return the client to the explorer based
// on the environment configured in the kernel arguments
func ExplorerClient() (*client.Client, error) {
	env, err := environment.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse node environment")
	}

	kp, err := identity.LoadKeyPair(seedPath)
	if err != nil {
		return nil, err
	}

	return Explorer(env.BcdbURL, kp)
}

func Explorer(url string, kp identity.KeyPair) (*client.Client, error) {
	return client.NewClient(url, nodeIdentity{kp})
}

type nodeIdentity struct {
	kp identity.KeyPair
}

func (n nodeIdentity) PrivateKey() ed25519.PrivateKey {
	return n.kp.PrivateKey
}

func (n nodeIdentity) Identity() string {
	return n.kp.Identity()
}
