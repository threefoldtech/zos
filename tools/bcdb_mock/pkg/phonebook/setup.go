package phonebook

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	phonebook "github.com/threefoldtech/zos/tools/bcdb_mock/pkg/phonebook/types"
	"go.mongodb.org/mongo-driver/mongo"
)

// Setup injects and initializes directory package
func Setup(parent *mux.Router, db *mongo.Database) error {
	if err := phonebook.Setup(context.TODO(), db); err != nil {
		return err
	}

	var userAPI UserAPI
	users := parent.PathPrefix("/users").Subrouter()

	users.HandleFunc("", mw.AsHandlerFunc(userAPI.create)).Methods(http.MethodPost)
	users.HandleFunc("", mw.AsHandlerFunc(userAPI.list)).Methods(http.MethodGet)
	// .Queries(
	// 	"page", `{page:\d+}`,
	// 	"size", `{size:\d+}`,
	// 	"name", "",
	// )

	users.HandleFunc("/{user_id}", mw.AsHandlerFunc(userAPI.register)).Methods(http.MethodPut)

	return nil
}
