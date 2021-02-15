package substrate

import (
	"crypto/ed25519"
	"fmt"
	"net"

	"github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/pkg/errors"
)

// EntityProof struct
type EntityProof struct {
	EntityID  types.U64
	Signature []byte
}

// Twin struct
type Twin struct {
	ID       types.U32
	Address  types.AccountID
	PubKey   [32]byte
	Entities []EntityProof
}

//YggdrasilAddress get the yggdrasil address from twin public key
func (t *Twin) YggdrasilAddress() (net.IP, error) {
	pk := ed25519.PublicKey(t.PubKey[:])
	ygg := NewYggdrasilIdentity(pk)
	return ygg.Address()
}

func (s *substrateClient) GetTwin(id uint32) (*Twin, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}

	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(meta, "TfgridModule", "Twins", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	var twin Twin
	ok, err := s.cl.RPC.State.GetStorageLatest(key, &twin)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup twin")
	}

	if !ok {
		return nil, fmt.Errorf("twin not found")
	}

	return &twin, nil
}
