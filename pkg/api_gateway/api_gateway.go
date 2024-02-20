package apigateway

import (
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

func NewAPIGateway(substrateURL string, identity substrate.Identity) (pkg.APIGateway, error) {
	sub, err := substrate.NewManager(substrateURL).Substrate()
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

func (g *apiGateway) GetContract(id uint64) (*substrate.Contract, error) {
	return g.sub.GetContract(id)
}

func (g *apiGateway) GetContractIDByNameRegistration(name string) (uint64, error) {
	return g.sub.GetContractIDByNameRegistration(name)
}

func (g *apiGateway) GetFarm(id uint32) (*substrate.Farm, error) {
	return g.sub.GetFarm(id)
}

func (g *apiGateway) GetNode(id uint32) (*substrate.Node, error) {
	return g.sub.GetNode(id)
}

func (g *apiGateway) GetNodeByTwinID(twin uint32) (uint32, error) {
	return g.sub.GetNodeByTwinID(twin)
}

func (g *apiGateway) GetNodeContracts(node uint32) ([]types.U64, error) {
	return g.sub.GetNodeContracts(node)
}

func (g *apiGateway) GetNodeRentContract(node uint32) (uint64, error) {
	return g.sub.GetNodeRentContract(node)
}

func (g *apiGateway) GetNodes(farmID uint32) ([]uint32, error) {
	return g.sub.GetNodes(farmID)
}

func (g *apiGateway) GetPowerTarget(nodeID uint32) (power substrate.NodePower, err error) {
	return g.sub.GetPowerTarget(nodeID)
}

func (g *apiGateway) GetTwin(id uint32) (*substrate.Twin, error) {
	return g.sub.GetTwin(id)
}

func (g *apiGateway) GetTwinByPubKey(pk []byte) (uint32, error) {
	return g.sub.GetTwinByPubKey(pk)
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
