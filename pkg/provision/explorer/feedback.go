package explorer

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg/provision"
)

// Signer interface is used to sign reservation result before
// sending them to the explorer
type Signer interface {
	Sign(b []byte) ([]byte, error)
}

// committerProvisioner is an implementation of the provision.Feedbacker
// that u sends results to the TFExplorer: https://github.com/threefoldtech/tfexplorer
type committerProvisioner struct {
	inner     provision.Provisioner
	client    client.Workloads
	converter ResultConverterFunc
	signer    Signer
	nodeID    string
	strategy  backoff.BackOff
}

// NewCommitterProvisioner creates a provisioner that makes sure provision results is committed
// to the explorer
func NewCommitterProvisioner(client client.Workloads, converter ResultConverterFunc, signer Signer, nodeID string, inner provision.Provisioner) provision.Provisioner {
	strategy := backoff.NewExponentialBackOff()
	strategy.MaxInterval = 15 * time.Second
	strategy.MaxElapsedTime = 3 * time.Minute

	return &committerProvisioner{
		inner:     inner,
		client:    client,
		converter: converter,
		signer:    signer,
		nodeID:    nodeID,
		strategy:  strategy,
	}
}

func (p *committerProvisioner) Provision(ctx context.Context, reservation *provision.Reservation) (*provision.Result, error) {
	result, err := p.inner.Provision(ctx, reservation)
	if err != nil {
		// an error return by the provisioner is a pipeline error
		// a handler error is instead available on the result object
		return result, err
	}
	bytes, err := result.Bytes()
	if err != nil {
		return result, errors.Wrap(err, "committer: failed to get result bytes")
	}

	signed, err := p.signer.Sign(bytes)
	if err != nil {
		return result, errors.Wrap(err, "committer: failed to sign result")
	}

	result.Signature = hex.EncodeToString(signed)
	if err := p.send(result); err != nil {
		return result, errors.Wrap(err, "committer: failed to send result to explorer")
	}

	if result.State != provision.StateOk {
		if err := p.delete(reservation.ID); err != nil {
			return result, errors.Wrap(err, "committer: failed to mark reservation as deleted")
		}
	}

	return result, nil
}

func (p *committerProvisioner) Decommission(ctx context.Context, reservation *provision.Reservation) error {
	if err := p.inner.Decommission(ctx, reservation); err != nil {
		return err
	}

	if err := p.delete(reservation.ID); err != nil {
		return errors.Wrap(err, "committer: failed to mark reservation as deleted")
	}

	return nil
}

// Feedback implements provision.Feedbacker
func (p *committerProvisioner) send(r *provision.Result) error {
	wr, err := p.converter(*r)
	if err != nil {
		return fmt.Errorf("failed to convert result into schema type: %w", err)
	}

	return backoff.Retry(func() error {
		err := p.client.NodeWorkloadPutResult(p.nodeID, r.ID, *wr)
		if err == nil || errors.Is(err, client.ErrRequestFailure) {
			// we only retry if err is a request failure err.
			return err
		}

		// otherwise retrying won't fix it, so we can terminate
		return backoff.Permanent(err)
	}, p.strategy)
}

// Deleted implements provision.Feedbacker
func (p *committerProvisioner) delete(id string) error {
	return backoff.Retry(func() error {
		err := p.client.NodeWorkloadPutDeleted(p.nodeID, id)
		if err == nil || errors.Is(err, client.ErrRequestFailure) {
			// we only retry if err is a request failure err.
			return err
		}

		// otherwise retrying won't fix it, so we can terminate
		return backoff.Permanent(err)
	}, p.strategy)
}

// // UpdateStats implements provision.Feedbacker
// func (e *Feedback) UpdateStats(nodeID string, w directory.WorkloadAmount, u directory.ResourceAmount) error {
// 	return backoff.Retry(func() error {
// 		err := e.client.Directory.NodeUpdateUsedResources(nodeID, u, w)
// 		if err == nil || errors.Is(err, client.ErrRequestFailure) {
// 			// we only retry if err is a request failure err.
// 			return err
// 		}

// 		// otherwise retrying won't fix it, so we can terminate
// 		return backoff.Permanent(err)
// 	}, e.strategy)
// }
