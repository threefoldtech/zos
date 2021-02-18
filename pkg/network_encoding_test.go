package pkg

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

func TestNetResourceUnmarshal(t *testing.T) {
	input := `{
		"name": "testnetworkd",
		"net_id": "net1",
		"ip_range": "10.0.0.0/16",
		"node_id": "HAcDwf7oCWEbn7ME1W4j3ACfsUo5kUgJqhk5MEDkbKis",
		"subnet": "10.0.1.0/24",
		"wg_private_key": "988c1e12dd04e5878b4cf008569f7b7163e7f3b2b619d339753c841c07dd0d6daf0b4dbc0b16e6ba29b21e7b600af76766e41e46419b05f9480e296f7934e83243680d6b7ad91a79442cfcbaf3a4898c603f15a024c2086a266fd18d",
		"wg_listen_port": 6380
	}`

	r := strings.NewReader(input)
	network := Network{}
	err := json.NewDecoder(r).Decode(&network)
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Equal(NetID("net1"), network.NetID)
	assert.Equal("988c1e12dd04e5878b4cf008569f7b7163e7f3b2b619d339753c841c07dd0d6daf0b4dbc0b16e6ba29b21e7b600af76766e41e46419b05f9480e296f7934e83243680d6b7ad91a79442cfcbaf3a4898c603f15a024c2086a266fd18d", network.WGPrivateKeyPlain)
	assert.Equal("10.0.1.0/24", network.Subnet.String())

}

func TestEncodeDecode(t *testing.T) {
	network := &Network{
		NetID: NetID("test"),
		Network: zos.Network{
			Name: "supernet",
			NetworkIPRange: gridtypes.NewIPNet(net.IPNet{
				IP:   net.ParseIP("10.0.0.0"),
				Mask: net.CIDRMask(16, 32),
			}),
			Subnet: gridtypes.NewIPNet(net.IPNet{
				IP:   net.ParseIP("10.0.1.0"),
				Mask: net.CIDRMask(24, 32),
			}),
			Peers: []zos.Peer{
				{
					Subnet: gridtypes.NewIPNet(net.IPNet{
						IP:   net.ParseIP("10.0.2.0"),
						Mask: net.CIDRMask(24, 32),
					}),
					Endpoint: "172.20.0.90:6380",
					AllowedIPs: []gridtypes.IPNet{
						gridtypes.NewIPNet(net.IPNet{
							IP:   net.ParseIP("10.0.1.0"),
							Mask: net.CIDRMask(24, 32),
						}),
					},
				},
			},
		},
	}
	b, err := json.Marshal(network)
	require.NoError(t, err)
	fmt.Println(string(b))

	decoded := &Network{}
	err = json.Unmarshal(b, decoded)
	require.NoError(t, err)
	assert.Equal(t, network.Name, decoded.Name)
	assert.Equal(t, network.NetworkIPRange.String(), decoded.NetworkIPRange.String())
	assert.Equal(t, network.NetID, decoded.NetID)

	assert.Equal(t, network.Subnet.String(), decoded.Subnet.String())
	assert.Equal(t, network.WGPrivateKeyPlain, decoded.WGPrivateKeyPlain)
	require.Equal(t, len(network.Peers), len(decoded.Peers))

	ePeer := network.Peers[0]
	aPeer := decoded.Peers[0]

	assert.Equal(t, ePeer.Subnet.String(), aPeer.Subnet.String())
	assert.Equal(t, ePeer.Endpoint, aPeer.Endpoint)
	assert.Equal(t, ePeer.AllowedIPs, aPeer.AllowedIPs)
	assert.Equal(t, ePeer.WGPublicKey, aPeer.WGPublicKey)
}
