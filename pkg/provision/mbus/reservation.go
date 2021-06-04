package mbus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/mbus"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

func (a *WorkloadsMessagebus) CreateOrUpdate(ctx context.Context, message mbus.Message, create bool) (interface{}, mw.Response) {
	bytes, err := base64.RawStdEncoding.DecodeString(message.Data)
	if err != nil {
		return nil, mw.Error(err, 400)
	}

	var deployment gridtypes.Deployment
	if err := json.Unmarshal(bytes, &deployment); err != nil {
		return nil, mw.BadRequest(err)
	}

	if err := deployment.Valid(); err != nil {
		return nil, mw.BadRequest(err)
	}

	authorized := false
	for _, twinID := range message.Twin_src {
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

	err = action(ctx, deployment)

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
