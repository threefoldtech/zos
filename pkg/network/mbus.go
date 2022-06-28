package network

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// Network message bus api
type Network struct {
	mgr pkg.Networker
}

// NewNetworkMessageBus creates a new messagebus instance
func NewNetworkMessageBus(router rmb.Router, mgr pkg.Networker) *Network {

	api := &Network{
		mgr: mgr,
	}
	api.setup(router)
	return api
}

func (n *Network) hasPublicIPv6Handler(ctx context.Context, payload []byte) (interface{}, error) {
	return n.hasPublicIPv6(ctx), nil
}

func (n *Network) setup(router rmb.Router) {

	// network handlers
	sub := router.Subroute("network")
	sub.WithHandler("list_wg_ports", n.listPorts)
	sub.WithHandler("public_config_get", n.getPublicConfig)
	sub.WithHandler("interfaces", n.listInterfaces)
	sub.WithHandler("has_ipv6", n.hasPublicIPv6Handler)
}

func (n *Network) listPorts(ctx context.Context, _ []byte) (interface{}, error) {
	return n.mgr.WireguardPorts()
}

func (n *Network) hasPublicIPv6(ctx context.Context) interface{} {
	ipData, err := n.mgr.GetPublicIPv6Subnet()
	return ipData.IP != nil && err == nil
}
func (n *Network) listInterfaces(ctx context.Context, _ []byte) (interface{}, error) {

	results := make(map[string][]net.IP)
	type q struct {
		inf    string
		ns     string
		rename string
	}
	for _, i := range []q{{"zos", "", "zos"}, {"nygg6", "ndmz", "ygg"}} {
		ips, _, err := n.mgr.Addrs(i.inf, i.ns)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get ips for '%s' interface", i)
		}

		results[i.rename] = func() []net.IP {
			list := make([]net.IP, 0, len(ips))
			for _, item := range ips {
				ip := net.IP(item)
				list = append(list, ip)
			}

			return list
		}()
	}

	return results, nil
}

func (n *Network) getPublicConfig(ctx context.Context, _ []byte) (interface{}, error) {
	return n.mgr.GetPublicConfig()
}
