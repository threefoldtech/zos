package api

import (
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// Engine is engine interface
type Engine interface {
	Feed() chan<- gridtypes.Reservation
	Get(gridtypes.ID) (gridtypes.Reservation, error)
}

// API is provision engine API
type API struct {
	engine Engine
}

// New creates a new API instance on given gorilla router
func New(router *mux.Router, engine Engine) (*API, error) {
	api := &API{engine: engine}

	return api, api.setup(router)
}

func (a *API) nextID() (string, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
