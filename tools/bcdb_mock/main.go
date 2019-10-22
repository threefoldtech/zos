package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var listen string

func main() {
	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")

	flag.Parse()

	resStore, err := LoadNodeStore()
	if err != nil {
		log.Fatalln("error loading node store: %v", err)
	}
	farmStore, err := LoadfarmStore()
	if err != nil {
		log.Fatalln("error loading farm store: %v", err)
	}
	provStore, err := LoadProvisionStore()
	if err != nil {
		log.Fatalln("error loading provision store: %v", err)
	}

	// provStore = &provisionStore{Reservations: make([]*reservation, 0, 20)}

	defer func() {
		if err := resStore.Save(); err != nil {
			log.Printf("failed to save data: %v\n", err)
		}
		if err := farmStore.Save(); err != nil {
			log.Printf("failed to save data: %v\n", err)
		}
	}()

	router := mux.NewRouter()

	router.HandleFunc("/nodes", resStore.registerNode).Methods("POST")

	router.HandleFunc("/nodes/{node_id}", resStore.nodeDetail).Methods("GET")
	router.HandleFunc("/nodes/{node_id}/interfaces", resStore.registerIfaces).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/ports", resStore.registerPorts).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/configure_public", resStore.configurePublic).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/capacity", resStore.registerCapacity).Methods("POST")
	router.HandleFunc("/nodes/{node_id}/uptime", resStore.updateUptimeHandler).Methods("POST")
	router.HandleFunc("/nodes", resStore.listNodes).Methods("GET")

	router.HandleFunc("/farms", farmStore.registerFarm).Methods("POST")
	router.HandleFunc("/farms", farmStore.listFarm).Methods("GET")
	router.HandleFunc("/farms/{farm_id}", farmStore.getFarm).Methods("GET")

	// compatibility with gedis_http
	router.HandleFunc("/nodes/list", resStore.cockpitListNodes).Methods("POST")
	router.HandleFunc("/farms/list", farmStore.cockpitListFarm).Methods("POST")

	router.HandleFunc("/reservations/{node_id}", resStore.Requires("node_id", provStore.reserve)).Methods("POST")
	router.HandleFunc("/reservations/{node_id}/poll", resStore.Requires("node_id", provStore.poll)).Methods("GET")
	router.HandleFunc("/reservations/{id}", provStore.get).Methods("GET")
	router.HandleFunc("/reservations/{id}", provStore.putResult).Methods("PUT")
	router.HandleFunc("/reservations/{id}/deleted", provStore.putDeleted).Methods("PUT")
	router.HandleFunc("/reservations/{id}", provStore.delete).Methods("DELETE")

	log.Printf("start on %s\n", listen)
	r := handlers.LoggingHandler(os.Stderr, router)
	r = handlers.CORS()(r)

	s := &http.Server{
		Addr:    listen,
		Handler: r,
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go s.ListenAndServe()

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		log.Printf("error during server shutdown: %v\n", err)
	}
}
