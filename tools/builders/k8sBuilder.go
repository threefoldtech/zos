package builders

import (
	"encoding/json"
	"io"
	"net"

	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

// K8sBuilder is a struct that can build K8S's
type K8sBuilder struct {
	workloads.K8S
}

// NewK8sBuilder creates a new K8S builder
func NewK8sBuilder() *K8sBuilder {
	return &K8sBuilder{
		K8S: workloads.K8S{
			Size: 1,
		},
	}
}

// LoadK8sBuilder loads a k8s builder based on a file path
func LoadK8sBuilder(reader io.Reader) (*K8sBuilder, error) {
	k8s := workloads.K8S{}

	err := json.NewDecoder(reader).Decode(&k8s)
	if err != nil {
		return &K8sBuilder{}, err
	}

	return &K8sBuilder{K8S: k8s}, nil
}

// Save saves the K8S builder to an IO.Writer
func (k8s *K8sBuilder) Save(writer io.Writer) error {
	err := json.NewEncoder(writer).Encode(k8s.K8S)
	if err != nil {
		return err
	}
	return err
}

// Build returns the kubernetes
func (k8s *K8sBuilder) Build() workloads.K8S {
	return k8s.K8S
}

// WithNodeID sets the node ID to the K8S
func (k8s *K8sBuilder) WithNodeID(nodeID string) *K8sBuilder {
	k8s.K8S.NodeId = nodeID
	return k8s
}

// WithSize sets the size to the K8S
func (k8s *K8sBuilder) WithSize(size int64) *K8sBuilder {
	k8s.K8S.Size = size
	return k8s
}

// WithNetworkID sets the network id to the K8S
func (k8s *K8sBuilder) WithNetworkID(id string) *K8sBuilder {
	k8s.K8S.NetworkId = id
	return k8s
}

// WithIPAddress sets the ip address to the K8S
func (k8s *K8sBuilder) WithIPAddress(ip net.IP) *K8sBuilder {
	k8s.K8S.Ipaddress = ip
	return k8s
}

// WithClusterSecret sets the cluster secret to the K8S
func (k8s *K8sBuilder) WithClusterSecret(secret string) *K8sBuilder {
	k8s.K8S.ClusterSecret = secret
	return k8s
}

// WithMasterIPs sets the master IPs to the K8S
func (k8s *K8sBuilder) WithMasterIPs(ips []net.IP) *K8sBuilder {
	k8s.K8S.MasterIps = ips
	return k8s
}

// WithSSHKeys sets the ssh keys to the K8S
func (k8s *K8sBuilder) WithSSHKeys(sshKeys []string) *K8sBuilder {
	k8s.K8S.SshKeys = sshKeys
	return k8s
}

// WithStatsAggregator sets the stats aggregators to the K8S
func (k8s *K8sBuilder) WithStatsAggregator(aggregators []workloads.StatsAggregator) *K8sBuilder {
	k8s.K8S.StatsAggregator = aggregators
	return k8s
}
