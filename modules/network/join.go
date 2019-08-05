package network

import (
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

	aloc := allocator.NewIPAllocator(&set, store, 0)

	return aloc.Get(id, "eth0", nil)
}
