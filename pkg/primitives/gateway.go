package primitives

import (
	"context"
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func (p *Primitives) gwProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	// what we need to do:
	// - does this node support gateways ?
	// this can be validated by checking if we have a "public" namespace

	// - Validation of ownership of the name
	// this must be done against substrate. Make sure that same user (twin) owns the
	// name int he workload config

	// - make necessary calls to gateway daemon.
	return nil, fmt.Errorf("not implemented")
}

func (p *Primitives) wgDecommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	// - make necessary calls to gateway daemon
	return fmt.Errorf("not implemented")
}
