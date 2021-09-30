package substrate

import (
	"net"

	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

// EntityProof struct
type EntityProof struct {
	EntityID  types.U32
	Signature string
}

// Twin struct
type Twin struct {
	Versioned
	ID       types.U32
	Account  AccountID
	IP       string
	Entities []EntityProof
}

//IPAddress parse the twin IP as net.IP
func (t *Twin) IPAddress() net.IP {
	return net.ParseIP(t.IP)
}

// GetTwinByPubKey gets a twin with public key
func (s *Substrate) GetTwinByPubKey(pk []byte) (uint32, error) {
	cl, meta, err := s.pool.Get()
	if err != nil {
		return 0, err
	}

	key, err := types.CreateStorageKey(meta, "TfgridModule", "TwinIdByAccountID", pk, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create substrate query key")
	}
	var id types.U32
	ok, err := cl.RPC.State.GetStorageLatest(key, &id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to lookup entity")
	}

	if !ok || id == 0 {
		return 0, errors.Wrap(ErrNotFound, "twin not found")
	}

	return uint32(id), nil
}

// GetTwin gets a twin
func (s *Substrate) GetTwin(id uint32) (*Twin, error) {
	cl, meta, err := s.pool.Get()
	if err != nil {
		return nil, err
	}

	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(meta, "TfgridModule", "Twins", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	raw, err := cl.RPC.State.GetStorageRawLatest(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup entity")
	}

	if len(*raw) == 0 {
		return nil, errors.Wrap(ErrNotFound, "twin not found")
	}

	version, err := s.getVersion(*raw)
	if err != nil {
		return nil, err
	}

	var twin Twin

	switch version {
	case 1:
		if err := types.DecodeFromBytes(*raw, &twin); err != nil {
			return nil, errors.Wrap(err, "failed to load object")
		}
	default:
		return nil, ErrUnknownVersion
	}

	return &twin, nil
}

// CreateTwin creates a twin
func (s *Substrate) CreateTwin(identity *Identity, ip net.IP) (uint32, error) {
	cl, meta, err := s.pool.Get()
	if err != nil {
		return 0, err
	}

	c, err := types.NewCall(meta, "TfgridModule.create_twin", ip.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to create call")
	}

	if _, err := s.call(cl, meta, identity, c); err != nil {
		return 0, errors.Wrap(err, "failed to create twin")
	}

	return s.GetTwinByPubKey(identity.PublicKey)
}

// UpdateTwin updates a twin
func (s *Substrate) UpdateTwin(identity *Identity, ip net.IP) (uint32, error) {
	cl, meta, err := s.pool.Get()
	if err != nil {
		return 0, err
	}

	c, err := types.NewCall(meta, "TfgridModule.update_twin", ip.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to create call")
	}

	if _, err := s.call(cl, meta, identity, c); err != nil {
		return 0, errors.Wrap(err, "failed to update twin")
	}

	return s.GetTwinByPubKey(identity.PublicKey)
}
