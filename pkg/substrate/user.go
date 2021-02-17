package substrate

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/pkg/errors"
)

// User from substrate
type User struct {
	ID        types.U32
	Name      string
	CountryID types.U32
	CityID    types.U32
	Address   AccountID
}

func (s *substrateClient) GetUser(id uint32) (*User, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}

	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(meta, "TfgridModule", "Entities", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	var user User
	ok, err := s.cl.RPC.State.GetStorageLatest(key, &user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup user")
	}

	if !ok {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil

}
