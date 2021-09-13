package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (p *Primitives) gwFQDNProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	result := zos.GatewayFQDNResult{}
	var proxy zos.GatewayFQDNProxy
	if err := json.Unmarshal(wl.Data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway proxy from reservation: %w", err)
	}
	networker := stubs.NewNetworkerStub(p.zbus)
	cfg, err := networker.GetPublicConfig(ctx)
	if err != nil {
		return "", errors.Wrap(err, "gateway is not supported on this node")
	}
	if cfg.Domain != "" && strings.HasSuffix(proxy.FQDN, cfg.Domain) {
		return "", errors.New("can't create a fqdn workload with a subdomain of the gateway's managed domain")
	}
	backends := make([]string, len(proxy.Backends))
	for idx, backend := range proxy.Backends {
		backends[idx] = string(backend)
	}
	gateway := stubs.NewGatewayStub(p.zbus)
	err = gateway.SetFQDNProxy(ctx, wl.ID.String(), proxy.FQDN, backends, proxy.TLSPassthrough)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup fqdn proxy")
	}
	return result, nil
}

func (p *Primitives) gwFQDNDecommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	gateway := stubs.NewGatewayStub(p.zbus)
	if err := gateway.DeleteNamedProxy(ctx, wl.ID.String()); err != nil {
		return errors.Wrap(err, "failed to delete fqdn proxy")
	}
	return nil
}
