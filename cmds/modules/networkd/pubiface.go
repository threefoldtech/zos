package networkd

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network/types"
)

// ErrNoPubInterface is the error returns by ReadPubIface when no public
// interface is configured
var ErrNoPubInterface = errors.New("no public interface configured for this node")

func getExitInterface(dir client.Directory, nodeID string) (*types.PubIface, error) {
	schemaNode, err := dir.NodeGet(nodeID, false)
	if err != nil {
		return nil, err
	}

	node := types.NewNodeFromSchema(schemaNode)
	if node.PublicConfig == nil {
		return nil, ErrNoPubInterface
	}

	return node.PublicConfig, nil
}

func watchPubIface(ctx context.Context, nodeID pkg.Identifier, dir client.Directory, ifaceVersion int) <-chan *types.PubIface {
	var currentVersion = ifaceVersion

	ch := make(chan *types.PubIface)
	go func() {
		defer func() {
			close(ch)
		}()

		for {
			select {
			case <-time.After(time.Minute * 10):
			case <-ctx.Done():
				break
			}

			exitIface, err := getExitInterface(dir, nodeID.Identity())
			if err != nil {
				if err == ErrNoPubInterface {
					continue
				}
				log.Error().Err(err).Msg("failed to read public interface")
				continue
			}

			if exitIface.Version <= currentVersion {
				continue
			}
			log.Info().
				Int("current version", currentVersion).
				Int("received version", exitIface.Version).
				Msg("new version of the public interface configuration")
			currentVersion = exitIface.Version

			select {
			case ch <- exitIface:
			case <-ctx.Done():
				break
			}
		}
	}()
	return ch
}
