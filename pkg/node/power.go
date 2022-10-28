package node

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/mw"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	downTarget = "down"
)

type Elections interface {
	IsLeader() bool
}

type powerRequest struct {
	Leader uint32 `json:"leader"`
	Node   uint32 `json:"node"`
	Target string `json:"target"`
}

type PowerServer struct {
	cl  zbus.Client
	sub substrate.Manager

	farm      pkg.FarmID
	node      uint32
	sk        ed25519.PrivateKey
	ut        *Uptime
	listen    string
	elections Elections
}

func NewPowerServer(
	cl zbus.Client,
	sub substrate.Manager,
	farm pkg.FarmID,
	node uint32,
	sk ed25519.PrivateKey,
	ut *Uptime) (*PowerServer, error) {

	if err := enableWol(wolInterface); err != nil {
		return nil, err
	}

	return &PowerServer{
		cl:     cl,
		sub:    sub,
		listen: fmt.Sprintf(":%d", PowerServerPort),
		farm:   farm,
		node:   node,
		sk:     sk,
		ut:     ut,
	}, nil
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
	var request powerRequest
	if p.elections.IsLeader() {
		// I am a leader, i don't listen to anyone
		return nil, mw.Forbidden(fmt.Errorf("is a leader node"))
	}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&request); err != nil {
		return nil, mw.BadRequest(err)
	}

	leader := mw.TwinID(r.Context())
	// the leader id in the request header doesn't match the body
	if leader != request.Leader {
		return nil, mw.UnAuthorized(fmt.Errorf("invalid leader id in request"))
	}

	if request.Target != downTarget {
		return nil, mw.BadRequest(fmt.Errorf("unknown power target '%s'", request.Target))
	}

	sub, err := p.sub.Substrate()
	if err != nil {
		return nil, mw.Error(err)
	}

	defer sub.Close()
	leaderNode, err := sub.GetNode(leader)
	if err != nil {
		return nil, mw.Error(errors.Wrap(err, "failed to get leader node"))
	}

	if leaderNode.FarmID != types.U32(p.farm) {
		return nil, mw.UnAuthorized(fmt.Errorf("requesting node is not in the same farm"))
	}

	// TODO: set current state on chain before powering off.
	if err := p.shutdown(); err != nil {
		return nil, mw.Error(err)
	}

	return nil, mw.Accepted()
}

func (p *PowerServer) Start(ctx context.Context) error {
	router := mux.NewRouter()
	signer := mw.NewSigner(p.sk)

	go p.events(ctx)

	// always sign responses
	router.Handle("/self", signer.Action(p.self)).Methods("GET")
	authorized := router.PathPrefix("/").Subrouter()
	twins, err := provision.NewSubstrateTwins(p.sub)
	if err != nil {
		return err
	}
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

	err = server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "failed to start power manager handler")
	}

	return nil
}
