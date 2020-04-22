package builders

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

func TestLoadNetwork(t *testing.T) {
	input := `{
		"iprange": "10.0.0.0/16",
		"network_resources": [
			{
				"node_id": "HAcDwf7oCWEbn7ME1W4j3ACfsUo5kUgJqhk5MEDkbKis",
				"iprange": "10.0.1.0/24",
				"wireguard_private_key_encrypted": "988c1e12dd04e5878b4cf008569f7b7163e7f3b2b619d339753c841c07dd0d6daf0b4dbc0b16e6ba29b21e7b600af76766e41e46419b05f9480e296f7934e83243680d6b7ad91a79442cfcbaf3a4898c603f15a024c2086a266fd18d",
				"wireguard_public_key": "L+V9o0fNYkMVKNqsX7spBzD/9oSvxM/C7ZCZX1jLO3Q=",
				"wireguard_listen_port": 6380
			}
		]
	}`

	r := strings.NewReader(input)
	network := workloads.Network{}
	err := json.NewDecoder(r).Decode(&network)
	require.NoError(t, err)
	assert := assert.New(t)

	networkAsReader := strings.NewReader(input)
	networkBuilder, err := LoadNetworkBuilder(networkAsReader)
	require.NoError(t, err)

	assert.Equal(networkBuilder.Network.Iprange, network.Iprange)
	assert.Equal(networkBuilder.Network.NetworkResources, network.NetworkResources)

	reservationBuilder := NewReservationBuilder(nil, nil).AddNetwork(*networkBuilder)

	assert.Equal(len(reservationBuilder.Reservation.DataReservation.Networks), 1)
}

func TestBuildNetwork(t *testing.T) {
	input := `{
		"iprange": "10.0.0.0/16",
		"network_resources": [
			{
				"node_id": "HAcDwf7oCWEbn7ME1W4j3ACfsUo5kUgJqhk5MEDkbKis",
				"iprange": "10.0.1.0/24",
				"wireguard_private_key_encrypted": "988c1e12dd04e5878b4cf008569f7b7163e7f3b2b619d339753c841c07dd0d6daf0b4dbc0b16e6ba29b21e7b600af76766e41e46419b05f9480e296f7934e83243680d6b7ad91a79442cfcbaf3a4898c603f15a024c2086a266fd18d",
				"wireguard_public_key": "L+V9o0fNYkMVKNqsX7spBzD/9oSvxM/C7ZCZX1jLO3Q=",
				"wireguard_listen_port": 6380
			}
		]
	}`

	r := strings.NewReader(input)
	network := workloads.Network{}
	err := json.NewDecoder(r).Decode(&network)
	require.NoError(t, err)
	assert := assert.New(t)

	networkBuilder := NewNetworkBuilder("test")

	networkBuilder.WithIPRange(network.Iprange).WithNetworkResources(network.NetworkResources)

	assert.Equal(networkBuilder.Network.Iprange, network.Iprange)
	assert.Equal(networkBuilder.Network.NetworkResources, network.NetworkResources)

	reservationBuilder := NewReservationBuilder(nil, nil).AddNetwork(*networkBuilder)

	assert.Equal(len(reservationBuilder.Reservation.DataReservation.Networks), 1)
}
