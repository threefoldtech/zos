package workloads

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/workloads/types"
	"go.mongodb.org/mongo-driver/mongo"
)

// Setup injects and initializes directory package
func Setup(parent *mux.Router, db *mongo.Database) error {
	if err := types.Setup(context.TODO(), db); err != nil {
		return err
	}

	var api API
	reservations := parent.PathPrefix("/reservations").Subrouter()

	reservations.HandleFunc("", mw.AsHandlerFunc(api.create)).Methods(http.MethodPost)
	reservations.HandleFunc("/{res_id}", mw.AsHandlerFunc(api.get)).Methods(http.MethodGet)
	// users.HandleFunc("", mw.AsHandlerFunc(api.list)).Methods(http.MethodGet)
	// users.HandleFunc("/{user_id}", mw.AsHandlerFunc(userAPI.register)).Methods(http.MethodPut)
	// users.HandleFunc("/{user_id}", mw.AsHandlerFunc(userAPI.get)).Methods(http.MethodGet)
	// users.HandleFunc("/{user_id}/validate", mw.AsHandlerFunc(userAPI.validate)).Methods(http.MethodPost)

	return nil
}
