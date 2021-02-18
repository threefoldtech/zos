package networkd

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// ErrNoPubInterface is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPubInterface = errors.New("no public interface configured for this node")

func getExitInterface() (*types.PubIface, error) {
	//TODO: this should load the public config from a config file
	return nil, nil
}
