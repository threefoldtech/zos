package vmlight

import (
	"context"

	"github.com/threefoldtech/zos4/pkg/gridtypes"
	"github.com/threefoldtech/zos4/pkg/provision"
	"github.com/threefoldtech/zos4/pkg/stubs"
)

func (m *Manager) Pause(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	vm := stubs.NewVMModuleStub(m.zbus)

	if err := vm.Lock(ctx, wl.ID.String(), true); err != nil {
		return provision.UnChanged(err)
	}

	return provision.Paused()
}

func (m *Manager) Resume(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	vm := stubs.NewVMModuleStub(m.zbus)

	if err := vm.Lock(ctx, wl.ID.String(), false); err != nil {
		return provision.UnChanged(err)
	}

	return provision.Ok()
}
