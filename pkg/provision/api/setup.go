package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

//Setup setup routes (v1)
func (a *Workloads) setup(router *mux.Router) error {
	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}).Methods(http.MethodGet).Name("test")

	workloads := router.PathPrefix("/workloads").Subrouter()

	workloads.Path("/").HandlerFunc(mw.AsHandlerFunc(a.create)).Methods(http.MethodPost).Name("workload-create")
	workloads.Path("/{id}").HandlerFunc(mw.AsHandlerFunc(a.get)).Methods(http.MethodGet).Name("workload-get")
	workloads.Path("/{id}").HandlerFunc(mw.AsHandlerFunc(a.delete)).Methods(http.MethodDelete).Name("workload-delete")
	return nil
}
