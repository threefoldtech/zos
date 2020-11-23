package primitives

import (
	"context"

	"github.com/threefoldtech/zos/pkg/provision"
)

// PublicIPResult result returned by publicIP reservation
type PublicIPResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

func (p *Provisioner) publicIPProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.publicIPProvisionImpl(ctx, reservation)
}

func (p *Provisioner) publicIPProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result PublicIPResult, err error) {
	return PublicIPResult{}, nil
}

func (p *Provisioner) publicIPDecomission(ctx context.Context, reservation *provision.Reservation) error {
	return nil
}
