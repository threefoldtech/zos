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

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// NetworkID construct a network ID based on a userID and network name
func NetworkID(userID, name string) gridtypes.NetID {
	buf := bytes.Buffer{}
	buf.WriteString(userID)
	buf.WriteString(":")
	buf.WriteString(name)
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return gridtypes.NetID(string(b))
}

// networkProvision is entry point to provision a network
func (p *Primitives) networkProvisionImpl(ctx context.Context, wl *gridtypes.Workload) error {
	var nr gridtypes.Network
	if err := json.Unmarshal(wl.Data, &nr); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	if err := nr.Valid(); err != nil {
		return fmt.Errorf("validation of the network resource failed: %w", err)
	}

	nr.NetID = NetworkID(wl.User.String(), nr.Name)

	mgr := stubs.NewNetworkerStub(p.zbus)
	log.Debug().Str("network", fmt.Sprintf("%+v", nr)).Msg("provision network")

	_, err := mgr.CreateNR(nr)
	if err != nil {
		return errors.Wrapf(err, "failed to create network resource for network %s", nr.NetID)
	}

	return nil
}

func (p *Primitives) networkProvision(ctx context.Context, wl *gridtypes.Workload) (interface{}, error) {
	return nil, p.networkProvisionImpl(ctx, wl)
}

func (p *Primitives) networkDecommission(ctx context.Context, wl *gridtypes.Workload) error {
	mgr := stubs.NewNetworkerStub(p.zbus)

	var network gridtypes.Network
	if err := json.Unmarshal(wl.Data, network); err != nil {
		return fmt.Errorf("failed to unmarshal network from reservation: %w", err)
	}

	network.NetID = NetworkID(wl.User.String(), network.Name)

	if err := mgr.DeleteNR(network); err != nil {
		return fmt.Errorf("failed to delete network resource: %w", err)
	}
	return nil
}
