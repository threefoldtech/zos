package explorer

import (
	"fmt"

	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/directory"
	"github.com/threefoldtech/zos/pkg/provision"
)

type ExplorerFeedback struct {
	client    *client.Client
	converter provision.ResultConverterFunc
}

func NewExplorerFeedback(client *client.Client, converter provision.ResultConverterFunc) *ExplorerFeedback {
	return &ExplorerFeedback{
		client:    client,
		converter: converter,
	}
}

func (e *ExplorerFeedback) Feedback(nodeID string, r *provision.Result) error {
	wr, err := e.converter(*r)
	if err != nil {
		return fmt.Errorf("failed to convert result into schema type: %w")
	}

	return e.client.Workloads.WorkloadPutResult(nodeID, r.ID, *wr)
}

func (e *ExplorerFeedback) Deleted(nodeID, id string) error {
	return e.client.Workloads.WorkloadPutDeleted(nodeID, id)
}

func (e *ExplorerFeedback) UpdateStats(nodeID string, w directory.WorkloadAmount, u directory.ResourceAmount) error {
	return e.client.Directory.NodeUpdateUsedResources(nodeID, u, w)
}
