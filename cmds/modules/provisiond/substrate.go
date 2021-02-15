package provisiond

import (
	"crypto/ed25519"
	"strconv"

	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/substrate"
)

type substrateUsers struct {
	sub substrate.Substrate
	mem *lru.Cache
}

// NewSubstrateUsers creates a substrate users db
func NewSubstrateUsers(url string) (provision.Users, error) {
	sub, err := substrate.NewSubstrate(url)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New(1023)
	if err != nil {
		return nil, err
	}

	return &substrateUsers{
		sub: sub,
		mem: cache,
	}, nil
}

func (s *substrateUsers) GetKey(id gridtypes.ID) ed25519.PublicKey {
	if value, ok := s.mem.Get(id); ok {
		return value.(ed25519.PublicKey)
	}

	idUint, err := strconv.ParseUint(id.String(), 10, 32)
	if err != nil {
		log.Error().Stringer("id", id).Err(err).Msg("failure in user public key look up")
		return nil
	}

	user, err := s.sub.GetUser(uint32(idUint))
	if err != nil {
		log.Error().Stringer("id", id).Err(err).Msg("failure in user public key look up")
		return nil
	}

	key := ed25519.PublicKey(user.PubKey[:])
	s.mem.Add(id, key)
	return key
}
