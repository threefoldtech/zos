package stubs

import (
	"context"
	zbus "github.com/threefoldtech/zbus"
	pkg "github.com/threefoldtech/zos/pkg"
)

type MonitorStub struct {
	client zbus.Client
	module string
	object zbus.ObjectID
}

func NewMonitorStub(client zbus.Client) *MonitorStub {
	return &MonitorStub{
		client: client,
		module: "monitor",
		object: zbus.ObjectID{
			Name:    "monitor",
			Version: "0.0.1",
		},
	}
}

func (s *MonitorStub) CPU(ctx context.Context) (<-chan pkg.CPUTimesStat, error) {
	ch := make(chan pkg.CPUTimesStat)
	recv, err := s.client.Stream(ctx, s.module, s.object, "CPU")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.CPUTimesStat
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}

func (s *MonitorStub) Disks(ctx context.Context) (<-chan pkg.DisksIOCountersStat, error) {
	ch := make(chan pkg.DisksIOCountersStat)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Disks")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.DisksIOCountersStat
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}

func (s *MonitorStub) Memory(ctx context.Context) (<-chan pkg.VirtualMemoryStat, error) {
	ch := make(chan pkg.VirtualMemoryStat)
	recv, err := s.client.Stream(ctx, s.module, s.object, "Memory")
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		for event := range recv {
			var obj pkg.VirtualMemoryStat
			if err := event.Unmarshal(&obj); err != nil {
				panic(err)
			}
			ch <- obj
		}
	}()
	return ch, nil
}
