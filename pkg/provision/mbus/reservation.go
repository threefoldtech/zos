package mbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

type deleteOrGetArgs struct {
	TwinID       uint32
	DeploymentID uint32
}

// CreateOrUpdate creates or updates a workload based on a message from the message bus
func (a *WorkloadsMessagebus) CreateOrUpdate(ctx context.Context, payload []byte, create bool) (interface{}, mw.Response) {
	var deployment gridtypes.Deployment
	if err := json.Unmarshal(payload, &deployment); err != nil {
		return nil, mw.BadRequest(err)
	}

	if err := deployment.Valid(); err != nil {
		return nil, mw.BadRequest(err)
	}

	twinSrc, ok := ctx.Value("twinSrc").([]int)
	if !ok {
		return nil, mw.BadRequest(errors.New("failed to load twin source from context"))
	}

	authorized := false
	for _, twinID := range twinSrc {
		if twinID == int(deployment.TwinID) {
			authorized = true
		}
	}
	if !authorized {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid user id in request message"))
	}

	if err := deployment.Verify(a.engine.Twins()); err != nil {
		return nil, mw.UnAuthorized(err)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	action := a.engine.Provision
	if !create {
		action = a.engine.Update
	}

	err := action(ctx, deployment)

	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if errors.Is(err, provision.ErrDeploymentExists) {
		return nil, mw.Conflict(err)
	} else if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(err)
	} else if errors.Is(err, provision.ErrDeploymentUpgradeValidationError) {
		return nil, mw.BadRequest(err)
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

// Delete deletes a workload
func (a *WorkloadsMessagebus) Delete(ctx context.Context, payload []byte) (interface{}, mw.Response) {
	var args deleteOrGetArgs
	err := json.Unmarshal(payload, &args)
	if err != nil {
		return nil, mw.Error(err)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	twinID := mw.TwinID(ctx)
	if args.TwinID != twinID {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid twin id in request url doesn't match http signature"))
	}

	err = a.engine.Deprovision(ctx, args.TwinID, args.DeploymentID, "requested by user")
	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(err)
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

// Get gets a workload
func (a *WorkloadsMessagebus) Get(ctx context.Context, payload []byte) (interface{}, mw.Response) {
	var args deleteOrGetArgs
	err := json.Unmarshal(payload, &args)
	if err != nil {
		return nil, mw.Error(err)
	}

	twinID := mw.TwinID(ctx)
	if args.TwinID != twinID {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid twin id in request url doesn't match http signature"))
	}

	deployment, err := a.engine.Storage().Get(args.TwinID, args.DeploymentID)
	if errors.Is(err, provision.ErrDeploymentNotExists) {
		return nil, mw.NotFound(fmt.Errorf("workload not found"))
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return deployment, nil
}
