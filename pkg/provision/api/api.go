package api

import (
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Workloads is provision engine Workloads
type Workloads struct {
	engine provision.Engine
}

// New creates a new API instance on given gorilla router
func New(router *mux.Router, engine provision.Engine) (*Workloads, error) {
	api := &Workloads{engine: engine}

	return api, api.setup(router)
}

func (a *Workloads) nextID() (string, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
