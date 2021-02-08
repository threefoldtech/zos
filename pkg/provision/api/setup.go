package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

//Setup setup routes (v1)
func (a *API) setup(router *mux.Router) error {
	reservation := router.PathPrefix("/reservation").Subrouter()

	reservation.Path("/").HandlerFunc(AsHandlerFunc(a.createReservation)).Methods(http.MethodPost).Name("reservation-create")
	return nil
}
