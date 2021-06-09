package mbus

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision/mw"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func (a *WorkloadsMessagebus) listPorts(ctx context.Context) (interface{}, mw.Response) {
	ports, err := stubs.NewNetworkerStub(a.cl).WireguardPorts(ctx)
	if err != nil {
		return nil, mw.Error(err)
	}

	return ports, nil
}

func (a *WorkloadsMessagebus) listPublicIps() (interface{}, mw.Response) {
	storage := a.engine.Storage()
	// for efficiency this method should just find out configured public Ips.
	// but currently the only way to do this is by scanning the nft rules
	// a nother less efficient but good for now solution is to scan all
	// reservations and find the ones with public IPs.

	twins, err := storage.Twins()
	if err != nil {
		return nil, mw.Error(errors.Wrap(err, "failed to list twins"))
	}
	ips := make([]string, 0)
	for _, twin := range twins {
		deploymentsIDs, err := storage.ByTwin(twin)
		if err != nil {
			return nil, mw.Error(errors.Wrap(err, "failed to list twin deployment"))
		}
		for _, id := range deploymentsIDs {
			deployment, err := storage.Get(twin, id)
			if err != nil {
				return nil, mw.Error(errors.Wrap(err, "failed to load deployment"))
			}
			workloads := deployment.ByType(zos.PublicIPType)

			for _, workload := range workloads {
				if workload.Result.State != gridtypes.StateOk {
					continue
				}

				data, err := workload.WorkloadData()
				if err != nil {
					return nil, mw.Error(err)
				}

				ip, _ := data.(*zos.PublicIP)
				if ip.IP.IP != nil {
					ips = append(ips, ip.IP.String())
				}
			}
		}
	}

	return ips, nil
}

func (a *WorkloadsMessagebus) getPublicConfig(ctx context.Context) (interface{}, mw.Response) {
	cfg, err := stubs.NewNetworkerStub(a.cl).GetPublicConfig(ctx)
	if err != nil {
		return nil, mw.NotFound(err)
	}

	return cfg, nil
}

func (a *WorkloadsMessagebus) setPublicConfig(ctx context.Context, payload []byte) (interface{}, mw.Response) {
	var cfg pkg.PublicConfig

	if err := json.Unmarshal(payload, &cfg); err != nil {
		return nil, mw.BadRequest(err)
	}

	err := stubs.NewNetworkerStub(a.cl).SetPublicConfig(ctx, cfg)
	if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Created()
}
