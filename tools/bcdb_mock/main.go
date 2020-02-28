package main

import (
	"context"
	"flag"

	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/tools/bcdb_mock/mw"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/directory"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/phonebook"
	"github.com/threefoldtech/zos/tools/bcdb_mock/pkg/workloads"
)

type Pkg func(*mux.Router, *mongo.Database) error

var listen string

func main() {
	app.Initialize()

	flag.StringVar(&listen, "listen", ":8080", "listen address, default :8080")

	flag.Parse()

	db, err := mw.NewDatabaseMiddleware("testing", "mongodb://localhost:27017")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	router := mux.NewRouter()

	router.Use(db.Middleware)

	pkgs := []Pkg{
		phonebook.Setup,
		directory.Setup,
		workloads.Setup,
	}

	for _, pkg := range pkgs {
		if err := pkg(router, db.Database()); err != nil {
			log.Error().Err(err).Msg("failed to register package")
		}
	}

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
