package provision

import (
	"crypto/ed25519"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

type substrateTwins struct {
	mgr substrate.Manager
	mem *lru.Cache
}

// NewSubstrateTwins creates a substrate users db that implements the provision.Users interface.
func NewSubstrateTwins(sub substrate.Manager) (Twins, error) {
	cache, err := lru.New(1024)
	if err != nil {
		return nil, err
	}

	return &substrateTwins{
		mgr: sub,
		mem: cache,
	}, nil
}

// GetKey gets twins public key
func (s *substrateTwins) GetKey(id uint32) ([]byte, error) {
	if value, ok := s.mem.Get(id); ok {
		return value.([]byte), nil
	}
	sub, err := s.mgr.Substrate()
	if err != nil {
		return nil, err
	}
	defer sub.Close()

	user, err := sub.GetTwin(id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get user with id '%d'", id)
	}

	key := user.Account.PublicKey()
	s.mem.Add(id, key)
	return []byte(key), nil
}

type substrateAdmins struct {
	twin uint32
	pk   ed25519.PublicKey
}

// NewSubstrateAdmins creates a substrate twins db that implements the provision.Users interface.
// but it also make sure the user is an admin
func NewSubstrateAdmins(mgr substrate.Manager, farmID uint32) (Twins, error) {
	sub, err := mgr.Substrate()
	if err != nil {
		return nil, err
	}
	defer sub.Close()
	farm, err := sub.GetFarm(farmID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get farm")
	}

	twin, err := sub.GetTwin(uint32(farm.TwinID))
	if err != nil {
		return nil, err
	}
	return &substrateAdmins{
		twin: uint32(farm.TwinID),
		pk:   twin.Account.PublicKey(),
	}, nil
}

// GetKey gets twin public key if twin is valid admin
func (s *substrateAdmins) GetKey(id uint32) ([]byte, error) {
	if id != s.twin {
		return nil, fmt.Errorf("twin with id '%d' is not an admin", id)
	}

	return []byte(s.pk), nil
}
