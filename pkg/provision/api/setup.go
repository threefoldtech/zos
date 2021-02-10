package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

//Setup setup routes (v1)
func (a *Workloads) setup(router *mux.Router) error {
	//TODO: this will need more twiking later
	//so the user getter will use the grid db
	//plus some internal in-memory cache

	router.Use(
		mw.NewAuthMiddleware(mw.NewUserKeyGetter()),
	)

	workloads := router.PathPrefix("/workloads").Subrouter()

	workloads.Path("/").HandlerFunc(mw.AsHandlerFunc(a.create)).Methods(http.MethodPost).Name("workload-create")
	workloads.Path("/{id}").HandlerFunc(mw.AsHandlerFunc(a.get)).Methods(http.MethodGet).Name("workload-get")
	workloads.Path("/{id}").HandlerFunc(mw.AsHandlerFunc(a.delete)).Methods(http.MethodDelete).Name("workload-delete")
	return nil
}
