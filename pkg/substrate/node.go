package substrate

import (
	"crypto/ed25519"
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

type Resources struct {
	HRU types.U64
	SRU types.U64
	CRU types.U64
	MRU types.U64
}

type Location struct {
	Longitude string
	Latitude  string
}

type Role struct {
	IsNode    bool
	IsGateway bool
}

// Decode implementation for the enum type
func (p *Role) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		p.IsNode = true
	case 1:
		p.IsGateway = true
	default:
		return fmt.Errorf("unknown CertificateType value")
	}

	return nil
}

func (m Role) Encode(encoder scale.Encoder) (err error) {
	if m.IsNode {
		err = encoder.PushByte(0)
	} else if m.IsGateway {
		err = encoder.PushByte(1)
	}

	return
}

type PublicConfig struct {
	IPv4 string
	IPv6 string
	GWv4 string
	GWv6 string
}

type OptionPublicConfig struct {
	HasValue bool
	AsValue  PublicConfig
}

func (m OptionPublicConfig) Encode(encoder scale.Encoder) (err error) {
	var i byte
	if m.HasValue {
		i = 1
	}
	err = encoder.PushByte(i)
	if err != nil {
		return err
	}

	if m.HasValue {
		err = encoder.Encode(m.AsValue)
	}

	return
}

// Farm type
type Node struct {
	Versioned
	ID           types.U32
	FarmID       types.U32
	TwinID       types.U32
	Resources    Resources
	Location     Location
	CountryID    types.U32
	CityID       types.U32
	Address      AccountID
	Role         Role
	PublicConfig OptionPublicConfig
}

//GetNodeByPubKey by an SS58 address
func (s *Substrate) GetNodeByPubKey(pk []byte) (*Node, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}

	key, err := types.CreateStorageKey(meta, "TfgridModule", "NodesByPubkeyID", pk, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}
	var id types.U32
	ok, err := s.cl.RPC.State.GetStorageLatest(key, &id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup entity")
	}

	if !ok || id == 0 {
		return nil, fmt.Errorf("node not found")
	}

	return s.GetNode(uint32(id))
}

func (s *Substrate) GetNode(id uint32) (*Node, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get substrate meta")
	}

	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}
	key, err := types.CreateStorageKey(meta, "TfgridModule", "Nodes", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	return s.getNode(key)
}

func (s *Substrate) getNode(key types.StorageKey) (*Node, error) {
	raw, err := s.cl.RPC.State.GetStorageRawLatest(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup entity")
	}

	if len(*raw) == 0 {
		return nil, errors.Wrap(ErrNotFound, "node not found")
	}

	version, err := s.getVersion(*raw)
	if err != nil {
		return nil, err
	}

	var node Node

	switch version {
	case 0:
		fallthrough
	case 1:
		if err := types.DecodeFromBytes(*raw, &node); err != nil {
			return nil, errors.Wrap(err, "failed to load object")
		}
	default:
		fmt.Println("version:", version)
		return nil, ErrUnknownVersion
	}

	return &node, nil
}

func (s *Substrate) CreateNode(sk ed25519.PrivateKey, node Node) (*Node, error) {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}

	c, err := types.NewCall(meta, "TfgridModule.create_node", node)
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
