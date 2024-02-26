package apigateway

import (
	"errors"
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
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
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.CreateNode(g.identity, node)
}

func (g *apiGateway) CreateTwin(relay string, pk []byte) (uint32, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.CreateTwin(g.identity, relay, pk)
}

func (g *apiGateway) EnsureAccount(activationURL string, termsAndConditionsLink string, terminsAndConditionsHash string) (info substrate.AccountInfo, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.EnsureAccount(g.identity, activationURL, terminsAndConditionsHash, terminsAndConditionsHash)
}

func (g *apiGateway) GetContract(id uint64) (substrate.Contract, pkg.ZBusError) {
	var zbusError pkg.ZBusError
	contract, err := g.sub.GetContract(id)
	if contract == nil {
		contract = &substrate.Contract{}
	}
	if errors.Is(err, substrate.ErrNotFound) {
		zbusError.Err = err
		zbusError.Code = pkg.CodeNotFound
	} else if err != nil {
		zbusError.Err = err
		zbusError.Code = pkg.CodeGenericError
	}
	return *contract, zbusError
}

func (g *apiGateway) GetContractIDByNameRegistration(name string) (uint64, pkg.ZBusError) {
	var zbusError pkg.ZBusError
	contractID, err := g.sub.GetContractIDByNameRegistration(name)

	if errors.Is(err, substrate.ErrNotFound) {
		zbusError.Err = err
		zbusError.Code = pkg.CodeNotFound
	} else if err != nil {
		zbusError.Err = err
		zbusError.Code = pkg.CodeGenericError
	}
	return contractID, zbusError
}

func (g *apiGateway) GetFarm(id uint32) (substrate.Farm, error) {
	farm, err := g.sub.GetFarm(id)
	if farm == nil {
		farm = &substrate.Farm{}
	}
	return *farm, err
}

func (g *apiGateway) GetNode(id uint32) (substrate.Node, error) {
	node, err := g.sub.GetNode(id)
	if node == nil {
		node = &substrate.Node{}
	}
	return *node, err
}

func (g *apiGateway) GetNodeByTwinID(twin uint32) (uint32, pkg.ZBusError) {
	var zbusError pkg.ZBusError
	nodeID, err := g.sub.GetNodeByTwinID(twin)

	if errors.Is(err, substrate.ErrNotFound) {
		zbusError.Err = err
		zbusError.Code = pkg.CodeNotFound
	} else if err != nil {
		zbusError.Err = err
		zbusError.Code = pkg.CodeGenericError
	}
	return nodeID, zbusError
}

func (g *apiGateway) GetNodeContracts(node uint32) ([]types.U64, error) {
	return g.sub.GetNodeContracts(node)
}

func (g *apiGateway) GetNodeRentContract(node uint32) (uint64, pkg.ZBusError) {
	var zbusError pkg.ZBusError
	contractID, err := g.sub.GetNodeRentContract(node)

	if errors.Is(err, substrate.ErrNotFound) {
		zbusError.Err = err
		zbusError.Code = pkg.CodeNotFound
	} else if err != nil {
		zbusError.Err = err
		zbusError.Code = pkg.CodeGenericError
	}
	return contractID, zbusError
}

func (g *apiGateway) GetNodes(farmID uint32) ([]uint32, error) {
	return g.sub.GetNodes(farmID)
}

func (g *apiGateway) GetPowerTarget(nodeID uint32) (power substrate.NodePower, err error) {
	return g.sub.GetPowerTarget(nodeID)
}

func (g *apiGateway) GetTwin(id uint32) (substrate.Twin, error) {
	twin, err := g.sub.GetTwin(id)
	if twin == nil {
		twin = &substrate.Twin{}
	}
	return *twin, err
}

func (g *apiGateway) GetTwinByPubKey(pk []byte) (uint32, pkg.ZBusError) {
	var zbusError pkg.ZBusError
	twinID, err := g.sub.GetTwinByPubKey(pk)

	if errors.Is(err, substrate.ErrNotFound) {
		zbusError.Err = err
		zbusError.Code = pkg.CodeNotFound
	} else if err != nil {
		zbusError.Err = err
		zbusError.Code = pkg.CodeGenericError
	}
	return twinID, zbusError
}

func (g *apiGateway) Report(consumptions []substrate.NruConsumption) (types.Hash, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.Report(g.identity, consumptions)
}

func (g *apiGateway) SetContractConsumption(resources ...substrate.ContractResources) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetContractConsumption(g.identity, resources...)
}

func (g *apiGateway) SetNodePowerState(up bool) (hash types.Hash, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.SetNodePowerState(g.identity, up)
}

func (g *apiGateway) UpdateNode(node substrate.Node) (uint32, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.UpdateNode(g.identity, node)
}

func (g *apiGateway) UpdateNodeUptimeV2(uptime uint64, timestampHint uint64) (hash types.Hash, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sub.UpdateNodeUptimeV2(g.identity, uptime, timestampHint)
}
