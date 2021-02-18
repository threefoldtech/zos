package networkd

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// ErrNoPubInterface is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPubInterface = errors.New("no public interface configured for this node")

func getExitInterface(dir client.Directory, nodeID string) (*types.PubIface, error) {
	//TODO: this should load the public config from a config file
	return nil, nil
}
