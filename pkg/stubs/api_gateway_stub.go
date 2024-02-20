// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	tfchainclientgo "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	zbus "github.com/threefoldtech/zbus"
)

type APIGatewayStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewAPIGatewayStub(client zbus.Client) *APIGatewayStub {
	return &APIGatewayStub{
		client: client,
		module: "apiGateway",
		object: zbus.ObjectID{
			Name:    "apiGateway",
			Version: "0.0.1",
		},
	}
}

func (s *APIGatewayStub) CreateNode(ctx context.Context, arg0 tfchainclientgo.Identity, arg1 tfchainclientgo.Node) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateNode", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) CreateTwin(ctx context.Context, arg0 tfchainclientgo.Identity, arg1 string, arg2 []uint8) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "CreateTwin", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) EnsureAccount(ctx context.Context, arg0 tfchainclientgo.Identity, arg1 string, arg2 string, arg3 string) (ret0 tfchainclientgo.AccountInfo, ret1 error) {
	args := []interface{}{arg0, arg1, arg2, arg3}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "EnsureAccount", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetContract(ctx context.Context, arg0 uint64) (ret0 interface{}, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetContract", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetContractIDByNameRegistration(ctx context.Context, arg0 string) (ret0 uint64, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetContractIDByNameRegistration", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetFarm(ctx context.Context, arg0 uint32) (ret0 interface{}, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetFarm", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetNode(ctx context.Context, arg0 uint32) (ret0 interface{}, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNode", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetNodeByTwinID(ctx context.Context, arg0 uint32) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeByTwinID", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetNodeContracts(ctx context.Context, arg0 uint32) (ret0 []types.U64, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeContracts", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetNodeRentContract(ctx context.Context, arg0 uint32) (ret0 uint64, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodeRentContract", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetNodes(ctx context.Context, arg0 uint32) (ret0 []uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetNodes", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetPowerTarget(ctx context.Context, arg0 uint32) (ret0 tfchainclientgo.NodePower, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetPowerTarget", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetTwin(ctx context.Context, arg0 uint32) (ret0 interface{}, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetTwin", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) GetTwinByPubKey(ctx context.Context, arg0 []uint8) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "GetTwinByPubKey", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) SetContractConsumption(ctx context.Context, arg0 tfchainclientgo.Identity, arg1 ...tfchainclientgo.ContractResources) (ret0 error) {
	args := []interface{}{arg0}
	for _, argv := range arg1 {
		args = append(args, argv)
	}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetContractConsumption", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret0 = result.CallError()
	loader := zbus.Loader{}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) SetNodePowerState(ctx context.Context, arg0 tfchainclientgo.Identity, arg1 bool) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "SetNodePowerState", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) UpdateNode(ctx context.Context, arg0 tfchainclientgo.Identity, arg1 tfchainclientgo.Node) (ret0 uint32, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateNode", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}

func (s *APIGatewayStub) UpdateNodeUptimeV2(ctx context.Context, arg0 tfchainclientgo.Identity, arg1 uint64, arg2 uint64) (ret0 types.Hash, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "UpdateNodeUptimeV2", args...)
	if err != nil {
		panic(err)
	}
	result.PanicOnError()
	ret1 = result.CallError()
	loader := zbus.Loader{
		&ret0,
	}
	if err := result.Unmarshal(&loader); err != nil {
		panic(err)
	}
	return
}
