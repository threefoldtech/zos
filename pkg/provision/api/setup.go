package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

//Setup setup routes (v1)
func (a *Workloads) setup(router *mux.Router) error {
	reservation := router.PathPrefix("/reservation").Subrouter()

	reservation.Path("/").HandlerFunc(mw.AsHandlerFunc(a.create)).Methods(http.MethodPost).Name("reservation-create")
	return nil
}
