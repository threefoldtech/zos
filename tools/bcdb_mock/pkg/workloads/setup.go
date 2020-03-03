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
	reservations.HandleFunc("", mw.AsHandlerFunc(api.list)).Methods(http.MethodGet)
	reservations.HandleFunc("/{res_id:\\d+}", mw.AsHandlerFunc(api.get)).Methods(http.MethodGet)

	reservations.HandleFunc("/workloads/{node_id}", mw.AsHandlerFunc(api.workloads)).Queries("from", "{from:\\d+}").Methods(http.MethodGet)
	reservations.HandleFunc("/workloads/{gwid:\\d+-\\d+}", mw.AsHandlerFunc(api.workloadGet)).Methods(http.MethodGet)
	reservations.HandleFunc("/workloads/{gwid:\\d+-\\d+}/{node_id}", mw.AsHandlerFunc(api.workloadPutResult)).Methods(http.MethodPut)

	// router.HandleFunc("/reservations/{node_id}/poll", nodeStore.Requires("node_id", resStore.poll)).Methods("GET")
	// router.HandleFunc("/reservations/{id}", resStore.get).Methods("GET")
	// router.HandleFunc("/reservations/{id}", resStore.putResult).Methods("PUT")
	// router.HandleFunc("/reservations/{id}/deleted", resStore.putDeleted).Methods("PUT")
	// router.HandleFunc("/reservations/{id}", resStore.delete).Methods("DELETE")

	return nil
}
