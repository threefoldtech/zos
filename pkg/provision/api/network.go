package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision/mw"
	"github.com/threefoldtech/zos/pkg/stubs"
)

//Network api
type Network struct {
	cl zbus.Client
}

// NewNetworkAPI creates a new NetworkAPI
func NewNetworkAPI(router *mux.Router, cl zbus.Client) (*Network, error) {
	api := &Network{cl}
	return api, api.setup(router)
}

func (n *Network) setup(router *mux.Router) error {

	api := router.PathPrefix("/wireguard").Subrouter()
	api.Path("/").HandlerFunc(mw.AsHandlerFunc(n.listPorts)).Name("wireguard-list")

	return nil
}

func (n *Network) listPorts(request *http.Request) (interface{}, mw.Response) {
	ports, err := stubs.NewNetworkerStub(n.cl).WireguardPorts()
	if err != nil {
		return nil, mw.Error(err)
	}

	return ports, nil
}
