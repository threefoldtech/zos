package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func (a *API) createReservation(request *http.Request) (interface{}, Response) {
	var reservation gridtypes.Reservation
	if err := json.NewDecoder(request.Body).Decode(&reservation); err != nil {
		return nil, BadRequest(err)
	}

	id, err := a.nextID()
	if err != nil {
		return nil, Error(err)
	}
	reservation.ID = gridtypes.ID(id)
	ctx, cancel := context.WithTimeout(request.Context(), 3*time.Minute)
	defer cancel()

	//TODO: validation of user identity goes here. and if we will
	//accept his reservation
	select {
	case <-ctx.Done():
		return nil, Unavailable(ctx.Err())
	case a.engine.Feed() <- reservation:
		return id, Accepted()
	}
}
