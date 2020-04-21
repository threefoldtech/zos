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
	return &K8sBuilder{}
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

// WithNodeID sets the node ID to the K8S
func (k8s *K8sBuilder) WithNodeID(nodeID string) {
	k8s.K8S.NodeId = nodeID
}

// WithSize sets the size to the K8S
func (k8s *K8sBuilder) WithSize(size int64) {
	k8s.K8S.Size = size
}

// WithNetworkID sets the network id to the K8S
func (k8s *K8sBuilder) WithNetworkID(id string) {
	k8s.K8S.NetworkId = id
}

// WithIPAddress sets the ip address to the K8S
func (k8s *K8sBuilder) WithIPAddress(ip net.IP) {
	k8s.K8S.Ipaddress = ip
}

// WithClusterSecret sets the cluster secret to the K8S
func (k8s *K8sBuilder) WithClusterSecret(secret string) {
	k8s.K8S.ClusterSecret = secret
}

// WithMasterIPs sets the master IPs to the K8S
func (k8s *K8sBuilder) WithMasterIPs(ips []net.IP) {
	k8s.K8S.MasterIps = ips
}

// WithSSHKeys sets the ssh keys to the K8S
func (k8s *K8sBuilder) WithSSHKeys(sshKeys []string) {
	k8s.K8S.SshKeys = sshKeys
}

// WithStatsAggregator sets the stats aggregators to the K8S
func (k8s *K8sBuilder) WithStatsAggregator(aggregators []workloads.StatsAggregator) {
	k8s.K8S.StatsAggregator = aggregators
}
