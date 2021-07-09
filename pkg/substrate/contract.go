package substrate

import (
	"crypto/ed25519"
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v3/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
)

// ContractState enum
type ContractState struct {
	IsCreated  bool
	IsDeployed bool
}

// Decode implementation for the enum type
func (r *ContractState) Decode(decoder scale.Decoder) error {
	b, err := decoder.ReadOneByte()
	if err != nil {
		return err
	}

	switch b {
	case 0:
		r.IsCreated = true
	case 1:
		r.IsDeployed = true
	default:
		return fmt.Errorf("unknown CertificateType value")
	}

	return nil
}

// Contract structure
type Contract struct {
	Versioned
	ContractID         types.U64
	TwinID             types.U32
	Node               AccountID
	Data               []byte
	DeploymentHash     string
	PublicIPsCount     types.U32
	State              ContractState
	LastUpdated        types.U64
	PreviousNUReported types.U64
	PublicIPs          []PublicIP
}

// GetContract we should not have calls to create contract, instead only get
func (s *Substrate) GetContract(id uint64) (*Contract, error) {
	bytes, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrap(err, "substrate: encoding error building query arguments")
	}

	key, err := types.CreateStorageKey(s.meta, "SmartContractModule", "Contracts", bytes, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create substrate query key")
	}

	return s.getContract(key)
}

func (s *Substrate) getContract(key types.StorageKey) (*Contract, error) {
	raw, err := s.cl.RPC.State.GetStorageRawLatest(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup contract")
	}

	if len(*raw) == 0 {
		return nil, errors.Wrap(ErrNotFound, "contract not found")
	}

	version, err := s.getVersion(*raw)
	if err != nil {
		return nil, err
	}

	var node Contract

	switch version {
	case 1:
		if err := types.DecodeFromBytes(*raw, &node); err != nil {
			return nil, errors.Wrap(err, "failed to load object")
		}
	default:
		return nil, ErrUnknownVersion
	}

	return &node, nil
}

// Consumption structure
type Consumption struct {
	Twin uint32         `json:"twin"`
	ID   uint64         `json:"id"`
	CRU  gridtypes.Unit `json:"cru"`
	SRU  gridtypes.Unit `json:"sru"`
	HRU  gridtypes.Unit `json:"hru"`
	MRU  gridtypes.Unit `json:"mru"`
	NRU  gridtypes.Unit `json:"nru"`
}

// Report structure
type Report struct {
	Timestamp   int64 `json:"timestamp"`
	Consumption []Consumption
}

// Report send a capacity report to substrate
func (s *Substrate) Report(sk ed25519.PrivateKey, report Report) error {
	c, err := types.NewCall(s.meta, "SmartContractModule.add_reports", report)
	if err != nil {
		return errors.Wrap(err, "failed to create call")
	}

	if err := s.call(sk, c); err != nil {
		return errors.Wrap(err, "failed to create report")
	}

	return nil
}
