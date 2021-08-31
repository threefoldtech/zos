package primitives

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func validateNameContract(twinID uint32, name string) error {
	// TODO: validate against substrate?
	return nil
}

func getNodeDomain() (string, error) {
	// TODO: how to get
	return "omar.com", nil
}

func (p *Primitives) gwProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {

	var proxy zos.GatewayNameProxy
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway proxy from reservation: %w", err)
	}
	baseDomain, err := getNodeDomain()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get domain base domain")
	}
	fqdn := fmt.Sprintf("%s.%s", proxy.Name, baseDomain)
	backends := make([]string, len(proxy.Backends))
	for idx, backend := range proxy.Backends {
		backends[idx] = string(backend)
	}
	// what we need to do:
	// - does this node support gateways ?
	// this can be validated by checking if we have a "public" namespace
	deployment := provision.GetDeployment(ctx)
	twinID := deployment.TwinID
	if err := validateNameContract(twinID, proxy.Name); err != nil {
		return nil, errors.Wrap(err, "failed to validate name contract")
	}
	// - Validation of ownership of the name (later)
	// this must be done against substrate. Make sure that same user (twin) owns the
	// name int he workload config

	// - make necessary calls to gateway daemon.
	// gateway := stubs.NewGatewayStub(p.zbus)
	// gateway.SetNamedProxy(ctx context.Context, arg0 string, arg1 []string)
	gateway := stubs.NewGatewayStub(p.zbus)
	if err := gateway.SetNamedProxy(ctx, fqdn, backends); err != nil {
		return nil, errors.Wrap(err, "failed to setup name proxy")
	}
	return nil, nil
}

func (p *Primitives) wgDecommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	deployment := provision.GetDeployment(ctx)
	twinID := deployment.TwinID
	var proxy zos.GatewayNameProxy
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return fmt.Errorf("failed to unmarshal gateway proxy from reservation: %w", err)
	}
	if err := validateNameContract(twinID, proxy.Name); err != nil {
		return errors.Wrap(err, "failed to validate name contract")
	}

	baseDomain, err := getNodeDomain()
	if err != nil {
		return errors.Wrap(err, "failed to get domain base domain")
	}
	fqdn := fmt.Sprintf("%s.%s", proxy.Name, baseDomain)

	gateway := stubs.NewGatewayStub(p.zbus)
	if err := gateway.DeleteNamedProxy(ctx, fqdn); err != nil {
		return errors.Wrap(err, "failed to delete name proxy")
	}
	return nil
}
