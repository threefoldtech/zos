package phonebook

import (
	"context"

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

	users.HandleFunc("", mw.AsHandlerFunc(userAPI.create)).Methods("POST")
	users.HandleFunc("/{user_id}", mw.AsHandlerFunc(userAPI.register)).Methods("PUT")
	return nil
}
