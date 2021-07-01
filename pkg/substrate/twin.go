package substrate

import (
	"crypto/ed25519"
	"fmt"
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

func (s *Substrate) GetTwinsByPubKey(pk []byte) ([]uint32, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}

	key, err := types.CreateStorageKey(meta, "TfgridModule", "TwinsByPubkey", pk, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}
	var ids []types.U32
	ok, err := s.cl.RPC.State.GetStorageLatest(key, &ids)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup entity")
	}

	if !ok {
		return nil, fmt.Errorf("node not found")
	}

	results := make([]uint32, 0, len(ids))
	for _, id := range ids {
		results = append(results, uint32(id))
	}

	return results, nil
}
func (s *Substrate) GetTwin(id uint32) (*Twin, error) {
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

	raw, err := s.cl.RPC.State.GetStorageRawLatest(key)
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

func (s *Substrate) CreateTwin(sk ed25519.PrivateKey, twin Twin) (*Node, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	c, err := types.NewCall(meta, "TfgridModule.create_twin", twin)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create call")
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := s.cl.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get genesisHash")
	}

	rv, err := s.cl.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return nil, err
	}

	identity, err := s.Identity(sk)
	if err != nil {
		return nil, err
	}

	//node.Address =identity.PublicKey
	account, err := s.getAccount(identity, meta)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get account")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(account.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: 1,
	}

	err = s.sign(&ext, identity, o)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign")
	}

	// Send the extrinsic
	sub, err := s.cl.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to submit extrinsic")
	}

	defer sub.Unsubscribe()

	for event := range sub.Chan() {
		if event.IsFinalized {
			break
		}
	}

	result, err := s.GetNodeByPubKey(identity.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node from chain, probably failed to create")
	}

	return result, nil
}
