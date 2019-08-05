package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zosv2/modules"
)

func TestAllocate(t *testing.T) {
	_, prefix, err := net.ParseCIDR("2a02:1802:5e:3105::/64")
	require.NoError(t, err)

	_, err = allocateIP("test", modules.NetID("my-network"), &modules.NetResource{Prefix: prefix}, "/tmp/leases")

	require.NoError(t, err)
}
