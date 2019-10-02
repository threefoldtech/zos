package network

import (
	"errors"
	"net"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// TNoDB define the interface to implement
// to talk to a Tenant Network object database
type TNoDB interface {
	GetFarm(farm pkg.Identifier) (*Farm, error)

	PublishInterfaces(node pkg.Identifier, ifaces []types.IfaceInfo) error
	GetNode(pkg.Identifier) (*types.Node, error)
	PublishWGPort(node pkg.Identifier, ports []uint) error

	SetPublicIface(node pkg.Identifier, pub *types.PubIface) error
	GetPubIface(node pkg.Identifier) (*types.PubIface, error)
}

// TNoDBUtils define the interface to implement
// to talk to a Tenant Network object database including utils methods
type TNoDBUtils interface {
	TNoDB
	RegisterAllocation(farm pkg.Identifier, allocation *net.IPNet) error
	RequestAllocation(farm pkg.Identifier) (*net.IPNet, *net.IPNet, uint8, error)

	SelectExitNode(node pkg.Identifier) error

	GetNetwork(netID pkg.NetID) (*pkg.Network, error)
	GetNetworksVersion(nodeID pkg.Identifier) (versions map[pkg.NetID]uint32, err error)
}

// Farm hold the ID, name and list of possible exit node of a farm
type Farm struct {
	ID        string   `json:"farm_id"`
	Name      string   `json:"name"`
	ExitNodes []string `json:"exit_nodes"`
}

// ErrNoPubIface is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPubIface = errors.New("no public interface configured for this node")
