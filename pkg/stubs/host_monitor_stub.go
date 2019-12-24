package stubs

import (
	"context"
	semver "github.com/blang/semver"
	zbus "github.com/threefoldtech/zbus"
)

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

func (s *HostMonitorStub) Version(ctx context.Context) (<-chan semver.Version, error) {
	ch := make(chan semver.Version)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Version")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj semver.Version
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}
