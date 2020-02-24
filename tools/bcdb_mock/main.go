package main

import (
	"context"
	"flag"

	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	"github.com/threefoldtech/zos/tools/bcdb_mock/types"
)

var listen string

func main() {
	app.Initialize()

	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")

	flag.Parse()

	nodeStore, err := loadNodeStore()
	if err != nil {
		log.Fatal().Err(err).Msg("error loading node store")
	}
	var farmAPI FarmAPI

	resStore, err := loadProvisionStore()
	if err != nil {
		log.Fatal().Err(err).Msg("error loading provision store")
	}

	// defer func() {
	// 	if err := nodeStore.Save(); err != nil {
	// 		log.Error().Err(err).Msg("failed to save data")
	// 	}
	// 	if err := farmStore.Save(); err != nil {
	// 		log.Printf("failed to save data: %v\n", err)
	// 	}
	// 	if err := resStore.Save(); err != nil {
	// 		log.Printf("failed to save reservations: %v\n", err)
	// 	}
	// }()

	db, err := mw.NewDatabaseMiddleware("testing", "mongodb://localhost:27017")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	// Setup all the models
	log.Info().Msg("setting up index")
	types.Setup(context.TODO(), db.Database())
	log.Info().Msg("setting up index completed")

	router := mux.NewRouter()

	router.Use(db.Middleware)
	router.HandleFunc("/farms", AsHandlerFunc(farmAPI.registerFarm)).Methods("POST")
	router.HandleFunc("/farms", AsHandlerFunc(farmAPI.listFarm)).Methods("GET")
	router.HandleFunc("/farms/{farm_id}", AsHandlerFunc(farmAPI.getFarm)).Methods("GET")

	router.HandleFunc("/nodes", nodeStore.registerNode).Methods("POST")

	router.HandleFunc("/nodes/{node_id}", nodeStore.nodeDetail).Methods("GET")
	router.HandleFunc("/nodes/{node_id}/interfaces", nodeStore.registerIfaces).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/ports", nodeStore.registerPorts).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/configure_public", nodeStore.configurePublic).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/capacity", nodeStore.registerCapacity).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/uptime", nodeStore.updateUptimeHandler).Methods("POST")
	router.HandleFunc("/nodes", nodeStore.listNodes).Methods("GET")
	router.HandleFunc("/nodes/{node_id}/used_resources", nodeStore.updateUsedResources).Methods("PUT")

	// compatibility with gedis_http
	router.HandleFunc("/nodes/list", nodeStore.cockpitListNodes).Methods("POST")
	router.HandleFunc("/farms/list", farmAPI.cockpitListFarm).Methods("POST")

	router.HandleFunc("/reservations/{node_id}", nodeStore.Requires("node_id", resStore.reserve)).Methods("POST")
	router.HandleFunc("/reservations/{node_id}/poll", nodeStore.Requires("node_id", resStore.poll)).Methods("GET")
	router.HandleFunc("/reservations/{id}", resStore.get).Methods("GET")
	router.HandleFunc("/reservations/{id}", resStore.putResult).Methods("PUT")
	router.HandleFunc("/reservations/{id}/deleted", resStore.putDeleted).Methods("PUT")
	router.HandleFunc("/reservations/{id}", resStore.delete).Methods("DELETE")

	log.Printf("start on %s\n", listen)
	r := handlers.LoggingHandler(os.Stderr, router)
	r = handlers.CORS()(r)

	s := &http.Server{
		Addr:    listen,
		Handler: r,
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go s.ListenAndServe()

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("error during server shutdown: %v\n", err)
	}
}
