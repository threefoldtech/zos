package stubs

import (
	zbus "github.com/threefoldtech/zbus"
	modules "github.com/threefoldtech/zosv2/modules"
)

type ContainerModuleStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewContainerModuleStub(client zbus.Client) *ContainerModuleStub {
	return &ContainerModuleStub{
		client: client,
		module: "container",
		object: zbus.ObjectID{
			Name:    "container",
			Version: "0.0.1",
		},
	}
}

func (s *ContainerModuleStub) Delete(arg0 string, arg1 modules.ContainerID) (ret0 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Delete", args...)
	if err != nil {
		panic(err)
	}
	ret0 = new(zbus.RemoteError)
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	return
}

func (s *ContainerModuleStub) Inspect(arg0 string, arg1 modules.ContainerID) (ret0 modules.ContainerInfo, ret1 error) {
	args := []interface{}{arg0, arg1}
	result, err := s.client.Request(s.module, s.object, "Inspect", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	ret1 = new(zbus.RemoteError)
	if err := result.Unmarshal(1, &ret1); err != nil {
		panic(err)
	}
	return
}

func (s *ContainerModuleStub) Run(arg0 string, arg1 string, arg2 string, arg3 []string, arg4 []string, arg5 modules.NetworkInfo, arg6 []modules.MountInfo, arg7 string) (ret0 modules.ContainerID, ret1 error) {
	args := []interface{}{arg0, arg1, arg2, arg3, arg4, arg5, arg6, arg7}
	result, err := s.client.Request(s.module, s.object, "Run", args...)
	if err != nil {
		panic(err)
	}
	if err := result.Unmarshal(0, &ret0); err != nil {
		panic(err)
	}
	ret1 = new(zbus.RemoteError)
	if err := result.Unmarshal(1, &ret1); err != nil {
		panic(err)
	}
	return
}
