package main

import (
	"context"
	"log"

	"github.com/threefoldtech/zbus"
)

func main() {
	server, err := zbus.NewRedisServer("server", "tcp://localhost:6379", 1)
	if err != nil {
		log.Fatalf("fail to connect to message broker server: %v\n", err)
	}

	var f flistModule
	server.Register(zbus.ObjectID{Name: "flist", Version: "0.0.1"}, &f)
	if err := server.Run(context.Background()); err != nil {
		log.Printf("unexpected error: %v\n", err)
	}
}
