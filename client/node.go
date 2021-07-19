package client

import (
	"context"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/rmb"
)

// NodeClient struct
type NodeClient struct {
	nodeTwin uint32
	bus      rmb.Client
}

type args map[string]interface{}

// NewNodeClient creates a new node RMB client. This client then can be used to
// communicate with the node over RMB.
func NewNodeClient(nodeTwin uint32, bus rmb.Client) *NodeClient {
	return &NodeClient{nodeTwin, bus}
}

// DeploymentDeploy sends the deployment to the node for processing.
func (n *NodeClient) DeploymentDeploy(ctx context.Context, dl gridtypes.Deployment) error {
	const cmd = "zos.deployment.deploy"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

// DeploymentUpdate update the given deployment. deployment must be a valid update for
// a deployment that has been already created via DeploymentDeploy
func (n *NodeClient) DeploymentUpdate(ctx context.Context, dl gridtypes.Deployment) error {
	const cmd = "zos.deployment.update"
	return n.bus.Call(ctx, n.nodeTwin, cmd, dl, nil)
}

// DeploymentGet gets a deployment via contract ID
func (n *NodeClient) DeploymentGet(ctx context.Context, contractID uint64) (dl gridtypes.Deployment, err error) {
	const cmd = "zos.deployment.get"
	in := args{
		"contract_id": contractID,
	}

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, in, &dl); err != nil {
		return dl, err
	}

	return dl, nil
}

// DeploymentDelete deletes a deployment, the node will make sure to decomission all deployments
// and set all workloads to deleted. A call to Get after delete is valid
func (n *NodeClient) DeploymentDelete(ctx context.Context, contractID uint64) error {
	const cmd = "zos.deployment.delete"
	in := args{
		"contract_id": contractID,
	}

	return n.bus.Call(ctx, n.nodeTwin, cmd, in, nil)
}

// Counters returns some node statistics. Including total and available cpu, memory, storage, etc...
func (n *NodeClient) Counters(ctx context.Context) (total gridtypes.Capacity, used gridtypes.Capacity, err error) {
	const cmd = "zos.statistics.get"
	var result struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}
	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return
	}

	return result.Total, result.Used, nil
}

// NetworkListWGPorts return a list of all "taken" ports on the node. A new deployment
// should be careful to use a free port for its network setup.
func (n *NodeClient) NetworkListWGPorts(ctx context.Context) ([]uint16, error) {
	const cmd = "zos.network.list_wg_ports"
	var result []uint16

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NetworkListIPs list taken public IPs on the node
func (n *NodeClient) NetworkListIPs(ctx context.Context) ([]string, error) {
	const cmd = "zos.network.list_public_ips"
	var result []string

	if err := n.bus.Call(ctx, n.nodeTwin, cmd, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NetworkGetPublicConfig returns the current public node network configuration. A node with a
// public config can be used as an access node for wireguard.
func (n *NodeClient) NetworkGetPublicConfig(ctx context.Context) (cfg pkg.PublicConfig, err error) {
	const cmd = "zos.network.public_config_get"

	if err = n.bus.Call(ctx, n.nodeTwin, cmd, nil, &cfg); err != nil {
		return
	}

	return
}
