package node

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/mw"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

type Elections interface {
	IsLeader() bool
}

type PowerServer struct {
	farm pkg.FarmID
	node uint32
	sk   ed25519.PrivateKey
	//ut     *Uptime
	listen string
	cl     zbus.Client
}

func NewPowerServer(
	cl zbus.Client,
	listen string,
	farm pkg.FarmID,
	node uint32,
	sk ed25519.PrivateKey) *PowerServer {

	return &PowerServer{
		cl:     cl,
		listen: listen,
		farm:   farm,
		node:   node,
		sk:     sk,
	}
}

func (p *PowerServer) self(r *http.Request) (interface{}, mw.Response) {
	stub := stubs.NewNetworkerStub(p.cl)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cfg, err := stub.GetPublicConfig(ctx)
	if err != nil {
		return nil, mw.Error(err)
	}

	type Self struct {
		ID      uint32     `json:"id"`
		Farm    pkg.FarmID `json:"farm"`
		Address string     `json:"address"`
		Access  bool       `json:"access"`
	}

	return Self{
		ID:      p.node,
		Farm:    p.farm,
		Address: hex.EncodeToString(p.sk.Public().(ed25519.PublicKey)),
		Access:  !cfg.IsEmpty(),
	}, nil
}

func (p *PowerServer) power(r *http.Request) (interface{}, mw.Response) {
	var request struct {
	}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&request); err != nil {
		return nil, mw.BadRequest(err)
	}

	return request, nil
}

func (p *PowerServer) Start(ctx context.Context, twins provision.Twins) error {
	router := mux.NewRouter()
	signer := mw.NewSigner(p.sk)

	// always sign responses
	router.Handle("/self", signer.Action(p.self)).Methods("GET")
	authorized := router.PathPrefix("/").Subrouter()
	auth := mw.NewAuthMiddleware(twins)
	authorized.Use(auth.Middleware)
	authorized.Handle("/power", signer.Action(p.power)).Methods("POST")

	server := http.Server{
		Addr:    p.listen,
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()

	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to start power manager handler")
	}

	return nil
}
