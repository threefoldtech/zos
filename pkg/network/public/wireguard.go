package public

import (
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/threefoldtech/zos/pkg/network/wireguard"
)

// NewWireguard creates a new wireguard instance in public namespace
// if exits
func NewWireguard(name string) (wg *wireguard.Wireguard, err error) {
	f := func(_ ns.NetNS) error {
		wg, err = wireguard.New(name)
		return err
	}

	ns := getPublicNamespace()
	if ns != nil {
		defer ns.Close()
		return wg, ns.Do(f)
	}

	return wg, f(nil)
}
