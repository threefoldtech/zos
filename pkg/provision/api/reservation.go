package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

func (a *Workloads) create(request *http.Request) (interface{}, mw.Response) {
	var reservation gridtypes.Workload
	if err := json.NewDecoder(request.Body).Decode(&reservation); err != nil {
		return nil, mw.BadRequest(err)
	}

	id, err := a.nextID()
	if err != nil {
		return nil, mw.Error(err)
	}
	reservation.ID = gridtypes.ID(id)
	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Minute)
	defer cancel()

	err = a.engine.Provision(ctx, reservation)
	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return id, mw.Accepted()
}
