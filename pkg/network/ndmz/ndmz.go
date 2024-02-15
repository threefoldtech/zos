package ndmz

import (
	"context"
	"net"

	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/vishvananda/netlink"
)

const (
	// FamilyAll get all IP address families
	FamilyAll = netlink.FAMILY_ALL
	//FamilyV4 gets all IPv4 addresses
	FamilyV4 = netlink.FAMILY_V4
	//FamilyV6 gets all IPv6 addresses
	FamilyV6 = netlink.FAMILY_V6
)

// DMZ is an interface used to create an DMZ network namespace
type DMZ interface {
	Namespace() string
	// create the ndmz network namespace and all requires network interfaces
	Create(ctx context.Context) error
	// delete the ndmz network namespace and clean up all network interfaces
	Delete() error
	// link a network resource from a user network to ndmz
	AttachNR(networkID, nr, ipamLeaseDir string) error

	DetachNR(networkID, ipamLeaseDir string) error
	// GetIP gets ndmz public ips from ndmz
	GetIP(family int) ([]net.IPNet, error)

	// Get gateway to given destination ip
	GetDefaultGateway(destination net.IP) (net.IP, error)

	// GetIPFor get the ip of an
	GetIPFor(inf string) ([]net.IPNet, error)
	//GetIP(family netlink.FAM)
	// SupportsPubIPv4 indicates if the node supports public ipv4 addresses for
	// workloads
	SupportsPubIPv4() bool

	//Interfaces information about dmz interfaces
	Interfaces() ([]types.IfaceInfo, error)
}
