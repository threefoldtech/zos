package provisiond

import (
	"context"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/events"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision"
)

type ContractEventHandler struct {
	node           uint32
	pool           substrate.Manager
	engine         provision.Engine
	eventsConsumer *events.RedisConsumer
}

func NewContractEventHandler(node uint32, mgr substrate.Manager, engine provision.Engine, events *events.RedisConsumer) ContractEventHandler {
	return ContractEventHandler{node: node, pool: mgr, engine: engine, eventsConsumer: events}
}

func (r *ContractEventHandler) current() (map[gridtypes.DeploymentID]gridtypes.Deployment, error) {
	// we need to build a list of all supposedly active contracts on this node
	storage := r.engine.Storage()
	_, deployments, err := storage.Capacity()
	if err != nil {
		return nil, err
	}

	running := make(map[gridtypes.DeploymentID]gridtypes.Deployment)
	for _, active := range deployments {
		running[active.DeploymentID] = active
	}

	return running, nil
}

func (r *ContractEventHandler) sync(ctx context.Context) error {
	log.Debug().Msg("synchronizing contracts with the chain")

	active, err := r.current()
	if err != nil {
		return errors.Wrap(err, "failed to get current active contracts")
	}
	sub, err := r.pool.Substrate()
	if err != nil {
		return err
	}

	defer sub.Close()

	onchain, err := sub.GetCapacityReservationContracts(r.node)
	if err != nil {
		return errors.Wrap(err, "failed to get active node contracts")
	}

	onChainContracts := make(map[types.U64]*substrate.Contract)
	for _, contractID := range onchain {
		contract, err := sub.GetContract(contractID)
		if errors.Is(err, substrate.ErrNotFound) {
			continue
		} else if err != nil {
			return errors.Wrapf(err, "failed to get contract: %d", contractID)
		}

		onChainContracts[contract.ContractID] = contract
	}
	// runningDeployments will eventually contain all runningDeployments contracts
	// that also exist on the chain.
	runningDeployments := make(map[gridtypes.DeploymentID]gridtypes.Deployment)
	deploymentsContracts := make(map[gridtypes.DeploymentID]types.U64)

	for _, contract := range onChainContracts {
		for _, deploymentID := range contract.ContractType.CapacityReservationContract.Deployments {
			// if contract is in active map, move it to running
			if dl, ok := active[gridtypes.DeploymentID(deploymentID)]; ok {
				runningDeployments[dl.DeploymentID] = dl
				deploymentsContracts[dl.DeploymentID] = contract.ContractID
			}

			// clean the active contracts from the map
			delete(active, gridtypes.DeploymentID(deploymentID))
		}
	}

	// the active map now contains all contracts that are active on the node
	// but not active on the chain (don't exist on chain anymore)
	// hence we can simply deprovision
	for _, dl := range active {
		// those contracts exits on node but not on chain.
		// need to be deleted
		if err := r.engine.Deprovision(ctx, dl.TwinID, dl.DeploymentID, "contract not active on chain"); err != nil {
			log.Error().Err(err).
				Uint32("twin", dl.TwinID).
				Uint64("deployment", dl.DeploymentID.U64()).
				Msg("failed to decommission contract")
		}
	}

	// the running map now contains all contracts that are still exist on the chain.
	// but some of those running contracts can be either locked, or unlocked.
	// hence we need to make sure deployments here has the same state as their contracts

	for id, dl := range runningDeployments {
		contract := onChainContracts[deploymentsContracts[id]]
		logger := log.With().
			Uint32("twin", dl.TwinID).
			Uint64("deployment", id.U64()).
			Uint64("contract", uint64(contract.ContractID)).
			Logger()

		// locked is chain state for that contract
		locked := contract.State.IsGracePeriod
		logger.Info().Bool("paused", locked).Msg("contract pause state")
		if locked == r.isLocked(&dl) {
			continue
		}

		// state is different
		logger.Info().Bool("paused", locked).Msg("changing contract pause state")
		action := r.engine.Resume
		if locked {
			action = r.engine.Pause
		}

		if err := action(ctx, dl.TwinID, dl.DeploymentID); err != nil {
			log.Error().Err(err).Msg("failed to change contract state")
		}
	}

	log.Debug().Msg("synchronization complete")
	return nil
}

func (r *ContractEventHandler) isLocked(dl *gridtypes.Deployment) bool {
	for _, wl := range dl.Workloads {
		if wl.Result.State.IsAny(gridtypes.StatePaused) {
			return true
		}
	}

	return false
}

func (r *ContractEventHandler) handleLock(ctx context.Context, event *pkg.ContractLockedEvent) error {
	action := r.engine.Resume
	if event.Lock {
		action = r.engine.Pause
	}
	sub, err := r.pool.Substrate()
	if err != nil {
		return err
	}
	defer sub.Close()

	contract, err := sub.GetContract(uint64(event.Contract))
	if err != nil {
		return errors.Wrapf(err, "failed to get contract with id: %d")
	}

	for _, deployment := range contract.ContractType.CapacityReservationContract.Deployments {
		if err := action(ctx, event.TwinId, gridtypes.DeploymentID(deployment)); err != nil {
			log.Error().Err(err).
				Uint32("twin", event.TwinId).
				Uint64("deployment", uint64(deployment)).
				Bool("lock", event.Lock).
				Msg("failed to set deployment locking contract")
		}
	}

	return nil
}

// Run runs the reporter
func (r *ContractEventHandler) Run(ctx context.Context) error {
	// go over all user reservations
	// take into account the following:
	// every is in seconds.
	cancellation, err := r.eventsConsumer.ContractCancelled(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to register to node events")
	}

	locking, err := r.eventsConsumer.ContractLocked(ctx)
	if err != nil {
		return err
	}

	if err := r.sync(ctx); err != nil {
		return errors.Wrap(err, "failed to synchronize active contracts")
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			go func() {
				if err := r.sync(ctx); err != nil {
					log.Error().Err(err).Msg("failed to synchronize contracts with the chain")
				}
			}()
		case event := <-cancellation:
			log.Debug().Msgf("received a cancel contract event %+v", event)

			// otherwise we know what contract to be deleted
			if err := r.engine.Deprovision(ctx, event.TwinId, event.Deployment, "contract canceled event received"); err != nil {
				log.Error().Err(err).
					Uint32("twin", event.TwinId).
					Uint64("deployment", event.Deployment.U64()).
					Msg("failed to decomission contract")
			}
		case event := <-locking:
			if err := r.handleLock(ctx, &event); err != nil {
				log.Error().Err(err).
					Uint32("twin", event.TwinId).
					Uint64("contract", event.Contract.U64()).
					Bool("lock", event.Lock).
					Msg("failed to handle locking contract")
			}
		}
	}
}
