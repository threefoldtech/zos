package apigateway

import (
	"encoding/hex"
	"errors"
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg"
)

type apiGateway struct {
	sub      *substrate.Substrate
	mu       sync.Mutex
	identity substrate.Identity
}

func NewAPIGateway(substrateURL []string, identity substrate.Identity) (pkg.APIGateway, error) {
	sub, err := substrate.NewManager(substrateURL...).Substrate()
	if err != nil {
		return nil, err
	}
	return &apiGateway{sub: sub, mu: sync.Mutex{}, identity: identity}, nil
}

func (g *apiGateway) CreateNode(node substrate.Node) (uint32, error) {
	log.Debug().
		Str("method", "CreateNode").
		Uint32("twin id", uint32(node.TwinID)).
		Uint32("farm id", uint32(node.FarmID)).
		Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.CreateNode(g.identity, node)
}

func (g *apiGateway) CreateTwin(relay string, pk []byte) (uint32, error) {
	log.Debug().Str("method", "CreateTwin").Str("relay", relay).Str("pk", hex.EncodeToString(pk)).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.CreateTwin(g.identity, relay, pk)
}

func (g *apiGateway) EnsureAccount(activationURL string, termsAndConditionsLink string, termsAndConditionsHash string) (info substrate.AccountInfo, err error) {
	log.Debug().
		Str("method", "EnsureAccount").
		Str("activation url", activationURL).
		Str("terms and conditions link", termsAndConditionsLink).
		Str("terms and conditions hash", termsAndConditionsHash).
		Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.EnsureAccount(g.identity, activationURL, termsAndConditionsLink, termsAndConditionsHash)
}

func (g *apiGateway) GetContract(id uint64) (substrate.Contract, pkg.SubstrateError) {
	log.Trace().Str("method", "GetContract").Uint64("id", id).Msg("method called")
	var SubstrateError pkg.SubstrateError
	contract, err := g.sub.GetContract(id)
	if contract == nil {
		contract = &substrate.Contract{}
	}
	if errors.Is(err, substrate.ErrNotFound) {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeNotFound
	} else if err != nil {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeGenericError
	}
	return *contract, SubstrateError
}

func (g *apiGateway) GetContractIDByNameRegistration(name string) (uint64, pkg.SubstrateError) {
	log.Trace().Str("method", "GetContractIDByNameRegistration").Str("name", name).Msg("method called")
	var SubstrateError pkg.SubstrateError
	contractID, err := g.sub.GetContractIDByNameRegistration(name)

	if errors.Is(err, substrate.ErrNotFound) {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeNotFound
	} else if err != nil {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeGenericError
	}
	return contractID, SubstrateError
}

func (g *apiGateway) GetFarm(id uint32) (substrate.Farm, error) {
	log.Trace().Str("method", "GetFarm").Uint32("id", id).Msg("method called")
	farm, err := g.sub.GetFarm(id)
	if farm == nil {
		farm = &substrate.Farm{}
	}
	return *farm, err
}

func (g *apiGateway) GetNode(id uint32) (substrate.Node, error) {
	log.Trace().Str("method", "GetNode").Uint32("id", id).Msg("method called")
	node, err := g.sub.GetNode(id)
	if node == nil {
		node = &substrate.Node{}
	}
	return *node, err
}

func (g *apiGateway) GetNodeByTwinID(twin uint32) (uint32, pkg.SubstrateError) {
	log.Trace().Str("method", "GetNodeByTwinID").Uint32("twin", twin).Msg("method called")
	var SubstrateError pkg.SubstrateError
	nodeID, err := g.sub.GetNodeByTwinID(twin)

	if errors.Is(err, substrate.ErrNotFound) {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeNotFound
	} else if err != nil {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeGenericError
	}
	return nodeID, SubstrateError
}

func (g *apiGateway) GetNodeContracts(node uint32) ([]types.U64, error) {
	log.Trace().Str("method", "GetNodeContracts").Uint32("node", node).Msg("method called")
	return g.sub.GetNodeContracts(node)
}

func (g *apiGateway) GetNodeRentContract(node uint32) (uint64, pkg.SubstrateError) {
	log.Trace().Str("method", "GetNodeRentContract").Uint32("node", node).Msg("method called")
	var SubstrateError pkg.SubstrateError
	contractID, err := g.sub.GetNodeRentContract(node)

	if errors.Is(err, substrate.ErrNotFound) {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeNotFound
	} else if err != nil {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeGenericError
	}
	return contractID, SubstrateError
}

func (g *apiGateway) GetNodes(farmID uint32) ([]uint32, error) {
	log.Trace().Str("method", "GetNodes").Uint32("farm id", farmID).Msg("method called")
	return g.sub.GetNodes(farmID)
}

func (g *apiGateway) GetPowerTarget(nodeID uint32) (power substrate.NodePower, err error) {
	log.Trace().Str("method", "GetPowerTarget").Uint32("node id", nodeID).Msg("method called")
	return g.sub.GetPowerTarget(nodeID)
}

func (g *apiGateway) GetTwin(id uint32) (substrate.Twin, error) {
	log.Trace().Str("method", "GetTwin").Uint32("id", id).Msg("method called")
	twin, err := g.sub.GetTwin(id)
	if twin == nil {
		twin = &substrate.Twin{}
	}
	return *twin, err
}

func (g *apiGateway) GetTwinByPubKey(pk []byte) (uint32, pkg.SubstrateError) {
	log.Trace().Str("method", "GetTwinByPubKey").Str("pk", hex.EncodeToString(pk)).Msg("method called")
	var SubstrateError pkg.SubstrateError
	twinID, err := g.sub.GetTwinByPubKey(pk)

	if errors.Is(err, substrate.ErrNotFound) {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeNotFound
	} else if err != nil {
		SubstrateError.Err = err
		SubstrateError.Code = pkg.CodeGenericError
	}
	return twinID, SubstrateError
}

func (g *apiGateway) Report(consumptions []substrate.NruConsumption) (types.Hash, error) {
	contractIDs := make([]uint64, 0, len(consumptions))
	for _, v := range consumptions {
		contractIDs = append(contractIDs, uint64(v.ContractID))
	}
	log.Debug().Str("method", "Report").Uints64("contract ids", contractIDs).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.Report(g.identity, consumptions)
}

func (g *apiGateway) SetContractConsumption(resources ...substrate.ContractResources) error {
	contractIDs := make([]uint64, 0, len(resources))
	for _, v := range resources {
		contractIDs = append(contractIDs, uint64(v.ContractID))
	}
	log.Debug().Str("method", "SetContractConsumption").Uints64("contract ids", contractIDs).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetContractConsumption(g.identity, resources...)
}

func (g *apiGateway) SetNodePowerState(up bool) (hash types.Hash, err error) {
	log.Debug().Str("method", "SetNodePowerState").Bool("up", up).Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetNodePowerState(g.identity, up)
}

func (g *apiGateway) UpdateNode(node substrate.Node) (uint32, error) {
	log.Debug().Str("method", "UpdateNode").Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.UpdateNode(g.identity, node)
}

func (g *apiGateway) UpdateNodeUptimeV2(uptime uint64, timestampHint uint64) (hash types.Hash, err error) {
	log.Debug().
		Str("method", "UpdateNodeUptimeV2").
		Uint64("uptime", uptime).
		Uint64("timestamp hint", timestampHint).
		Msg("method called")
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.UpdateNodeUptimeV2(g.identity, uptime, timestampHint)
}
