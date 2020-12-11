package primitives

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"

	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// NetworkID construct a network ID based on a userID and network name
func NetworkID(userID, name string) pkg.NetID {
	buf := bytes.Buffer{}
	buf.WriteString(userID)
	buf.WriteString(name)
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return pkg.NetID(string(b))
}

// networkProvision is entry point to provision a network
func (p *Primitives) networkProvisionImpl(ctx context.Context, reservation *provision.Reservation) error {
	nr := pkg.NetResource{}
	if err := json.Unmarshal(reservation.Data, &nr); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	if err := nr.Valid(); err != nil {
		return fmt.Errorf("validation of the network resource failed: %w", err)
	}

	nr.NetID = NetworkID(reservation.User, nr.Name)

	mgr := stubs.NewNetworkerStub(p.zbus)
	log.Debug().Str("network", fmt.Sprintf("%+v", nr)).Msg("provision network")

	_, err := mgr.CreateNR(nr)
	if err != nil {
		return errors.Wrapf(err, "failed to create network resource for network %s", nr.NetID)
	}

	return nil
}

func (p *Primitives) networkProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, reservation)
}

func (p *Primitives) networkDecommission(ctx context.Context, reservation *provision.Reservation) error {
	mgr := stubs.NewNetworkerStub(p.zbus)

	network := &pkg.NetResource{}
	if err := json.Unmarshal(reservation.Data, network); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	network.NetID = NetworkID(reservation.User, network.Name)

	if err := mgr.DeleteNR(*network); err != nil {
		return fmt.Errorf("failed to delete network resource: %w", err)
	}
	return nil
}
