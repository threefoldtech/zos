// GENERATED CODE
// --------------
// please do not edit manually instead use the "zbusc" to regenerate

package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	"net"
)

type NetworkerLightStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewNetworkerLightStub(client zbus.Client) *NetworkerLightStub {
	return &NetworkerLightStub{
		client: client,
		module: "netlight",
		object: zbus.ObjectID{
			Name:    "netlight",
			Version: "0.0.1",
		},
	}
}

func (s *NetworkerLightStub) AttachMycelium(ctx context.Context, arg0 string, arg1 string, arg2 []uint8) (ret0 pkg.TapDevice, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "AttachMycelium", args...)
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

func (s *NetworkerLightStub) AttachPrivate(ctx context.Context, arg0 string, arg1 string, arg2 []uint8) (ret0 pkg.TapDevice, ret1 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "AttachPrivate", args...)
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

func (s *NetworkerLightStub) AttachZDB(ctx context.Context, arg0 string) (ret0 string, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "AttachZDB", args...)
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

func (s *NetworkerLightStub) Create(ctx context.Context, arg0 string, arg1 net.IPNet, arg2 []uint8) (ret0 error) {
	args := []interface{}{arg0, arg1, arg2}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Create", args...)
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

func (s *NetworkerLightStub) Delete(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Delete", args...)
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

func (s *NetworkerLightStub) Detach(ctx context.Context, arg0 string) (ret0 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Detach", args...)
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

func (s *NetworkerLightStub) Interfaces(ctx context.Context, arg0 string, arg1 string) (ret0 pkg.Interfaces, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "Interfaces", args...)
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

func (s *NetworkerLightStub) ZDBIPs(ctx context.Context, arg0 string) (ret0 [][]uint8, ret1 error) {
	args := []interface{}{arg0}
	result, err := s.client.RequestContext(ctx, s.module, s.object, "ZDBIPs", args...)
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
