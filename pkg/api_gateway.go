package pkg

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

//go:generate zbusc -module api-gateway -version 0.0.1 -name api-gateway -package stubs github.com/threefoldtech/zos/pkg+APIGateway stubs/api_gateway_stub.go

type APIGateway interface {
	CreateNode(node substrate.Node) (uint32, error)
	CreateTwin(relay string, pk []byte) (uint32, error)
	EnsureAccount(activationURL string, termsAndConditionsLink string, terminsAndConditionsHash string) (info substrate.AccountInfo, err error)
	GetContract(id uint64) (substrate.Contract, Error)
	GetContractIDByNameRegistration(name string) (uint64, Error)
	GetFarm(id uint32) (substrate.Farm, error)
	GetNode(id uint32) (substrate.Node, error)
	GetNodeByTwinID(twin uint32) (uint32, Error)
	GetNodeContracts(node uint32) ([]types.U64, error)
	GetNodeRentContract(node uint32) (uint64, Error)
	GetNodes(farmID uint32) ([]uint32, error)
	GetPowerTarget(nodeID uint32) (power substrate.NodePower, err error)
	GetTwin(id uint32) (substrate.Twin, error)
	GetTwinByPubKey(pk []byte) (uint32, Error)
	SetContractConsumption(resources ...substrate.ContractResources) error
	SetNodePowerState(up bool) (hash types.Hash, err error)
	UpdateNode(node substrate.Node) (uint32, error)
	UpdateNodeUptimeV2(uptime uint64, timestampHint uint64) (hash types.Hash, err error)
}

type Error struct {
	Err  error
	Code int
}

func (e *Error) IsError() bool {
	return e.Code != CodeNoError
}

func (e *Error) IsCode(codes ...int) bool {
	for _, code := range codes {
		if code == e.Code {
			return true
		}
	}
	return false
}

const (
	CodeGenericError = iota - 1
	CodeNoError
	CodeNotFound
)
