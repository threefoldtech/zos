package ndmz

import (
	"context"
	"net"

	"github.com/threefoldtech/zos/pkg/network/nr"
	"github.com/threefoldtech/zos/pkg/network/types"
)

const (
	//BridgeNDMZ is the name of the ipv4 routing bridge in the ndmz namespace
	BridgeNDMZ = "br-ndmz"
	//NetNSNDMZ name of the dmz namespace
	NetNSNDMZ = "ndmz"

	ndmzNsMACDerivationSuffix6 = "-ndmz6"
	ndmzNsMACDerivationSuffix4 = "-ndmz4"

	// DMZPub4 ipv4 public interface
	DMZPub4 = "npub4"
	// DMZPub6 ipv6 public interface
	DMZPub6 = "npub6"

	//nrPubIface is the name of the public interface in a network resource
	nrPubIface = "public"

	tonrsIface = "tonrs"
)

// DMZ is an interface used to create an DMZ network namespace
type DMZ interface {
	// create the ndmz network namespace and all requires network interfaces
	Create(ctx context.Context) error
	// delete the ndmz network namespace and clean up all network interfaces
	Delete() error
	// link a network resource from a user network to ndmz
	AttachNR(networkID string, nr *nr.NetResource, ipamLeaseDir string) error
	// Return the interface used by ndmz to router public ipv6 traffic
	IP6PublicIface() string
	// configure an address on the public IPv6 interface
	SetIP(net.IPNet) error
	// SupportsPubIPv4 indicates if the node supports public ipv4 addresses for
	// workloads
	SupportsPubIPv4() bool

	//IsIPv4Only checks if dmz is ipv4 only (no ipv6 support)
	IsIPv4Only() (bool, error)

	//Interfaces information about dmz interfaces
	Interfaces() ([]types.IfaceInfo, error)
}
