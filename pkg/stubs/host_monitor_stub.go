package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
	"time"
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

func (s *HostMonitorStub) IPs(ctx context.Context) (<-chan pkg.NetlinkAddresses, error) {
	ch := make(chan pkg.NetlinkAddresses)
	recv, err := s.client.Stream(ctx, s.module, s.object, "IPs")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.NetlinkAddresses
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}

func (s *HostMonitorStub) Uptime(ctx context.Context) (<-chan time.Duration, error) {
	ch := make(chan time.Duration)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Uptime")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj time.Duration
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}
