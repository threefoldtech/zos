package builders

import (
	"encoding/json"
	"io"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

// NetworkBuilder is a struct that can build networks
type NetworkBuilder struct {
	workloads.Network
}

// NewNetworkBuilder creates a new network builder
func NewNetworkBuilder(name string) *NetworkBuilder {
	return &NetworkBuilder{
		Network: workloads.Network{
			Name: name,
		},
	}
}

// LoadNetworkBuilder loads a network builder based on a file path
func LoadNetworkBuilder(reader io.Reader) (*NetworkBuilder, error) {
	network := workloads.Network{}

	err := json.NewDecoder(reader).Decode(&network)
	if err != nil {
		return &NetworkBuilder{}, err
	}

	return &NetworkBuilder{Network: network}, nil
}

// Save saves the network builder to an IO.Writer
func (n *NetworkBuilder) Save(writer io.Writer) error {
	err := json.NewEncoder(writer).Encode(n.Network)
	if err != nil {
		return err
	}
	return err
}

// Build returns the network
func (n *NetworkBuilder) Build() workloads.Network {
	return n.Network
}

// TODO ADD NODE ID TO NETWORK?

// WithName sets the ip range to the network
func (n *NetworkBuilder) WithName(name string) *NetworkBuilder {
	n.Network.Name = name
	return n
}

// WithIPRange sets the ip range to the network
func (n *NetworkBuilder) WithIPRange(ipRange schema.IPRange) *NetworkBuilder {
	n.Network.Iprange = ipRange
	return n
}

// WithStatsAggregator sets the stats aggregators to the network
func (n *NetworkBuilder) WithStatsAggregator(aggregators []workloads.StatsAggregator) *NetworkBuilder {
	n.Network.StatsAggregator = aggregators
	return n
}

// WithNetworkResources sets the network resources to the network
func (n *NetworkBuilder) WithNetworkResources(netResources []workloads.NetworkNetResource) *NetworkBuilder {
	n.Network.NetworkResources = netResources
	return n
}
