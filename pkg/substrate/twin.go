package substrate

import (
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
	Account  AccountID
	IP       string
	Entities []EntityProof
}

//IPAddress parse the twin IP as net.IP
func (t *Twin) IPAddress() net.IP {
	return net.ParseIP(t.IP)
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
