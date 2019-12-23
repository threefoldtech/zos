package stubs

import zbus "github.com/threefoldtech/zbus"

type HostMonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewHostMonitorStub(client zbus.Client) *HostMonitorStub {
	return &HostMonitorStub{
		client: client,
		module: "monitor",
		object: zbus.ObjectID{
			Name:    "host",
			Version: "0.0.1",
		},
	}
}
