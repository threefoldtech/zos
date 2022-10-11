package power

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

var (
	errConnectionError = fmt.Errorf("connection error")
)

type Manager struct {
	events *stubs.EventsStub
	farm   pkg.FarmID
	node   uint32
}

func NewManager(cl zbus.Client, farm pkg.FarmID, node uint32) *Manager {
	events := stubs.NewEventsStub(cl)
	return &Manager{
		events: events,
		farm:   farm,
		node:   node,
	}
}

func (m *Manager) listen(ctx context.Context) error {
	stream, err := m.events.PowerChangeEvent(ctx)
	if err != nil {
		return errors.Wrapf(errConnectionError, "failed to connect to zbus events: %s", err)
	}
	return nil
}
