package gedis

import (
	"net"
	"testing"

	"github.com/threefoldtech/zos/pkg/schema"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
	"github.com/threefoldtech/zos/pkg/network/types"
)

func TestNetworkPublishInterfaces(t *testing.T) {
	require := require.New(t)
	pool, conn := getTestPool()
	gedis := Gedis{
		pool:      pool,
		namespace: "default",
	}

	id := pkg.StrIdentifier("node-1")
	r := schema.MustParseIPRange("192.168.1.2/24")
	args := Args{
		"node_id": id,
		"ifaces": []directory.TfgridNodeIface1{
			{
				Name: "eth0",
				Addrs: []schema.IPRange{
					r,
				},
				Gateway: []net.IP{
					net.ParseIP("192.168.1.1"),
				},
			},
		},
	}

	conn.On("Do", "default.nodes.publish_interfaces", mustMarshal(t, args)).
		Return(mustMarshal(t, directory.TfgridNode2{
			NodeID: "node-1",
		}), nil)

	inf := types.IfaceInfo{
		Name: "eth0",
		Addrs: []*net.IPNet{
			&r.IPNet,
		},
		Gateway: []net.IP{net.ParseIP("192.168.1.1")},
	}
	err := gedis.PublishInterfaces(id, []types.IfaceInfo{inf})

	require.NoError(err)
	conn.AssertCalled(t, "Close")
}
