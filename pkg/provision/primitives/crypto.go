package primitives

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (p *Primitives) decryptSecret(ctx context.Context, user gridtypes.ID, secret string, version int) (string, error) {
	if len(secret) == 0 {
		return "", nil
	}

	engine := provision.GetEngine(ctx)

	identity := stubs.NewIdentityManagerStub(p.zbus)

	bytes, err := hex.DecodeString(secret)
	if err != nil {
		return "", err
	}

	var (
		out []byte
	)
	// now only one version is supported
	switch version {
	default:
		userPubKey := engine.Users().GetKey(user)
		if userPubKey == nil {
			return "", fmt.Errorf("failed to retrieve user %s public key", user)
		}
		out, err = identity.DecryptECDH(bytes, userPubKey)
	}

	return string(out), err
}

func fetchUserPublicKey(userID string) (ed25519.PublicKey, error) {
	iid, err := strconv.Atoi(userID)
	if err != nil {
		return nil, err
	}

	explorer, err := app.ExplorerClient()
	if err != nil {
		return nil, err
	}

	user, err := explorer.Phonebook.Get(schema.ID(iid))
	if err != nil {
		return nil, err
	}

	b, err := hex.DecodeString(user.Pubkey)
	if err != nil {
		return nil, err
	}

	return ed25519.PublicKey(b), nil
}
