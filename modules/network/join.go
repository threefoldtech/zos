package network

import (
	"net"
	"path"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/allocator"
	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend/disk"
	"github.com/threefoldtech/zosv2/modules"
)

func allocateIP(id string, netID modules.NetID, nr *modules.NetResource, storageRoot string) (*current.IPConfig, error) {
	storePath := path.Join(storageRoot, "lease")
	store, err := disk.New(string(netID), storePath)
	if err != nil {
		return nil, err
	}

	r := allocator.Range{
		Subnet: types.IPNet(*nr.Prefix),
	}

	if err := r.Canonicalize(); err != nil {
		return nil, err
	}

	set := allocator.RangeSet{r}

	// unfortunately, calling the allocator Get() directly will try to allocate
	// a new IP. if the ID/nic already has an ip allocated it will just fail instead of returning
	// the same IP.
	// So we have to check the store ourselves to see if there is already an IP allocated
	// to this container, and if one found, we return it.
	store.Lock()
	ips := store.GetByID(id, "eth0")
	store.Unlock()
	if len(ips) > 0 {
		ip := ips[0]
		rng, err := set.RangeFor(ip)
		if err != nil {
			return nil, err
		}

		return &current.IPConfig{
			Address: net.IPNet{IP: ip, Mask: rng.Subnet.Mask},
		}, nil
	}

	aloc := allocator.NewIPAllocator(&set, store, 0)

	return aloc.Get(id, "eth0", nil)
}
