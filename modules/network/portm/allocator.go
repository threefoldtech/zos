package portm

import (
	"errors"

	"github.com/threefoldtech/zosv2/modules/network/portm/backend"
)

var ErrNoFreePort = errors.New("no free port find")

type PortRange struct {
	Start int
	End   int
}

type allocator struct {
	pRange PortRange
	store  backend.Store
}

var _ PortAllocator = (*allocator)(nil)

func NewAllocator(pRange PortRange, store backend.Store) PortAllocator {
	return &allocator{
		pRange: pRange,
		store:  store,
	}
}

func (a *allocator) Reserve(ns string) (int, error) {

	allocatedPorts, err := a.store.GetByNS(ns)
	if err != nil {
		return 0, err
	}

	start, err := a.store.LastReserved(ns)
	if err != nil {
		return 0, err
	}

	if start == -1 || start >= a.pRange.End {
		// nothing reserve yet
		// or we reach the end of the range
		// then start from the start to re-use released port
		start = a.pRange.Start
	}

	for port := start; port <= a.pRange.End; port++ {
		if contains(allocatedPorts, port) {
			continue
		}

		reserved, err := a.store.Reserve(ns, port)
		if err != nil {
			return 0, err
		}
		if reserved {
			return port, nil
		}
	}

	return 0, ErrNoFreePort
}

func (a *allocator) Release(ns string, port int) error {
	return a.store.Release(ns, port)
}

func contains(s []int, port int) bool {
	for i := range s {
		if s[i] == port {
			return true
		}
	}
	return false
}
