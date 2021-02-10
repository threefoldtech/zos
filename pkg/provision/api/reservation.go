package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
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

	if err := reservation.Valid(); err != nil {
		return nil, mw.BadRequest(err)
	}

	//TODO: validate signature

	err = a.engine.Provision(ctx, reservation)
	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return id, mw.Accepted()
}

func (a *Workloads) delete(request *http.Request) (interface{}, mw.Response) {
	id := mux.Vars(request)["id"]
	if len(id) == 0 {
		return nil, mw.BadRequest(fmt.Errorf("invalid id format"))
	}

	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Minute)
	defer cancel()

	err := a.engine.Deprovision(ctx, gridtypes.ID(id), "requested by user")
	if err == context.DeadlineExceeded {
		return nil, mw.Unavailable(ctx.Err())
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (a *Workloads) get(request *http.Request) (interface{}, mw.Response) {
	id := mux.Vars(request)["id"]
	if len(id) == 0 {
		return nil, mw.BadRequest(fmt.Errorf("invalid id format"))
	}

	wl, err := a.engine.Storage().Get(gridtypes.ID(id))
	if err == provision.ErrWorkloadNotExists {
		return nil, mw.NotFound(fmt.Errorf("workload not found"))
	} else if err != nil {
		return nil, mw.Error(err)
	}

	return wl, nil
}
