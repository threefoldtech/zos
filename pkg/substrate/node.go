package substrate

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

var (
	errAccountNotFound = fmt.Errorf("account not found")
)

/*
curl --header "Content-Type: application/json" \
  --request POST \
  --data '{"kycSignature": "", "data": {"name": "", "email": ""}, "substrateAccountID": "5DAprR72N6s7AWGwN7TzV9MyuyGk9ifrq8kVxoXG9EYWpic4"}' \
  https://api.substrate01.threefold.io/activate
*/

func (s *Substrate) activateAccount(identity signature.KeyringPair) error {
	const url = "https://api.substrate01.threefold.io/activate"

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(map[string]string{
		"substrateAccountID": identity.Address,
	})

	response, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return errors.Wrap(err, "failed to call activation service")
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusConflict {
		// it went fine.
		return nil
	}

	return fmt.Errorf("failed to activate account: %s", response.Status)
}

func (s *Substrate) EnsureAccount(sk ed25519.PrivateKey) (info types.AccountInfo, err error) {
	identity, err := s.Identity(sk)
	if err != nil {
		return
	}
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return info, err
	}

	info, err = s.getAccount(identity, meta)
	if errors.Is(err, errAccountNotFound) {
		// account activation
		if err = s.activateAccount(identity); err != nil {
			return
		}

		// after activation this can take up to 10 seconds
		// before the account is actually there !

		exp := backoff.NewExponentialBackOff()
		exp.MaxElapsedTime = 10 * time.Second
		exp.MaxInterval = 3 * time.Second

		err = backoff.Retry(func() error {
			info, err = s.getAccount(identity, meta)
			return err
		}, exp)

		return
	}

	return

}

func (s *Substrate) Identity(sk ed25519.PrivateKey) (signature.KeyringPair, error) {
	str := types.HexEncodeToString(sk[:32])

	return signature.KeyringPairFromSecret(str, 0)
}

func (s *Substrate) getAccount(identity signature.KeyringPair, meta *types.Metadata) (info types.AccountInfo, err error) {
	key, err := types.CreateStorageKey(meta, "System", "Account", identity.PublicKey, nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create storage key")
		return
	}

	ok, err := s.cl.RPC.State.GetStorageLatest(key, &info)
	if err != nil || !ok {
		if !ok {
			return info, errAccountNotFound
		}

		return
	}

	return
}

func (s *Substrate) GetAccount(sk ed25519.PrivateKey) (info types.AccountInfo, err error) {
	identity, err := s.Identity(sk)
	if err != nil {
		return
	}
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return info, err
	}

	return s.getAccount(identity, meta)
}

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

func (s *Substrate) GetNodeByPubKey(pk ed25519.PublicKey) (*Node, error) {
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

func (s *Substrate) CreateNode(sk ed25519.PrivateKey, node Node) error {
	meta, err := s.cl.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}

	c, err := types.NewCall(meta, "TfgridModule.create_node", node)
	if err != nil {
		return errors.Wrap(err, "failed to create call")
	}

	// Create the extrinsic
	ext := types.NewExtrinsic(c)

	genesisHash, err := s.cl.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "failed to get genesisHash")
	}

	rv, err := s.cl.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return err
	}

	identity, err := s.Identity(sk)

	if err != nil {
		return err
	}

	//node.Address =identity.PublicKey
	account, err := s.getAccount(identity, meta)
	if err != nil {
		return errors.Wrap(err, "failed to get account")
	}

	o := types.SignatureOptions{
		// BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(account.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: 1,
	}

	err = ext.Sign(identity, o)
	if err != nil {
		return errors.Wrap(err, "failed to sign")
	}

	// Send the extrinsic
	_, err = s.cl.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "failed to submit extrinsic")
	}

	return nil
}
