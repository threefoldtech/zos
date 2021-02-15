package substrate

import (
	"crypto/ed25519"
	"strconv"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client"
	"github.com/centrifuge/go-substrate-rpc-client/types"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type substrateUsers struct {
	cl  *gsrpc.SubstrateAPI
	mem *lru.Cache
}

// NewSubstrateUsers creates a substrate users db
func NewSubstrateUsers(url string) (provision.Users, error) {
	cl, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New(1023)
	if err != nil {
		return nil, err
	}

	return &substrateUsers{
		cl:  cl,
		mem: cache,
	}, nil
}

func (s *substrateUsers) getKey(id gridtypes.ID) (ed25519.PublicKey, error) {

	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}
	idUint, err := strconv.ParseUint(id.String(), 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid user id format expecting uint")
	}
	bytes, err := types.EncodeToBytes(idUint)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(meta, "TfgridModule", "Entities", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	var user struct {
		ID        types.U64
		Name      string
		CountryID types.U64
		CityID    types.U64
		Address   types.AccountID
		PubKey    [32]byte
	}

	ok, err := s.cl.RPC.State.GetStorageLatest(key, &user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup user public key")
	}

	if !ok {
		return nil, errors.Wrap(err, "user not found")
	}

	return ed25519.PublicKey(user.PubKey[:]), nil

}

func (s *substrateUsers) GetKey(id gridtypes.ID) ed25519.PublicKey {
	if value, ok := s.mem.Get(id); ok {
		return value.(ed25519.PublicKey)
	}

	key, err := s.getKey(id)
	if err != nil {
		log.Error().Stringer("id", id).Err(err).Msg("failure in user public key look up")
	}

	return key
}
