package directory

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/tools/explorer/mw"
	directory "github.com/threefoldtech/zos/tools/explorer/pkg/directory/types"
	"go.mongodb.org/mongo-driver/mongo"
)

// Setup injects and initializes directory package
func Setup(parent *mux.Router, db *mongo.Database) error {
	if err := directory.Setup(context.TODO(), db); err != nil {
		return err
	}

	nkg := mw.NewNodeKeyGetter()
	ukg := mw.NewUserKeyGetter(db)

	var farmAPI FarmAPI
	farms := parent.PathPrefix("/farms").Subrouter()
	farmAuthenticated := parent.PathPrefix("/farms").Subrouter()
	farmAuthenticated.Use(func(h http.Handler) http.Handler {
		return mw.AuthMiddleware(db, farmAuthenticated, ukg)
	})

	farms.HandleFunc("", mw.AsHandlerFunc(farmAPI.registerFarm)).Methods("POST")
	farms.HandleFunc("", mw.AsHandlerFunc(farmAPI.listFarm)).Methods("GET")
	farms.HandleFunc("/{farm_id}", mw.AsHandlerFunc(farmAPI.getFarm)).Methods("GET")

	var nodeAPI NodeAPI
	nodes := parent.PathPrefix("/nodes").Subrouter()

	nodesAuthenticated := parent.PathPrefix("/nodes").Subrouter()
	userAuthenticated := parent.PathPrefix("/nodes").Subrouter()
	nodesAuthenticated.Use(func(h http.Handler) http.Handler {
		return mw.AuthMiddleware(db, nodesAuthenticated, nkg)
	})
	userAuthenticated.Use(func(h http.Handler) http.Handler {
		return mw.AuthMiddleware(db, nodesAuthenticated, ukg)
	})

	nodesAuthenticated.HandleFunc("", mw.AsHandlerFunc(nodeAPI.registerNode)).Methods("POST")
	nodes.HandleFunc("", mw.AsHandlerFunc(nodeAPI.listNodes)).Methods("GET")
	nodes.HandleFunc("/{node_id}", mw.AsHandlerFunc(nodeAPI.nodeDetail)).Methods("GET")
	nodesAuthenticated.HandleFunc("/{node_id}/interfaces", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerIfaces))).Methods("POST")
	nodesAuthenticated.HandleFunc("/{node_id}/ports", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerPorts))).Methods("POST")
	userAuthenticated.HandleFunc("/{node_id}/configure_public", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.configurePublic))).Methods("POST")
	nodesAuthenticated.HandleFunc("/{node_id}/capacity", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.registerCapacity))).Methods("POST")
	nodesAuthenticated.HandleFunc("/{node_id}/uptime", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.updateUptimeHandler))).Methods("POST")
	nodesAuthenticated.HandleFunc("/{node_id}/used_resources", mw.AsHandlerFunc(nodeAPI.Requires("node_id", nodeAPI.updateReservedResources))).Methods("POST")

	return nil
}
