package events

import (
	"context"
	"encoding/binary"
	"os"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	rpc "github.com/centrifuge/go-substrate-rpc-client/v4/gethrpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
)

// State is used to track the blocks already processed by the processor
type State interface {
	Set(num types.BlockNumber) error
	Get(cl *gsrpc.SubstrateAPI) (types.BlockNumber, error)
}

type FileState struct {
	last types.BlockNumber
	path string
}

func NewFileState(path string) State {
	return &FileState{path: path}
}

func (e *FileState) Set(num types.BlockNumber) error {
	e.last = num
	var d [4]byte
	binary.BigEndian.PutUint32(d[:], uint32(num))
	if err := os.WriteFile(e.path, d[:], 0644); err != nil {
		return errors.Wrap(err, "failed to commit last block state")
	}
	return nil
}

func (e *FileState) Get(cl *gsrpc.SubstrateAPI) (types.BlockNumber, error) {
	// getLatest need to
	if e.last != 0 {
		return e.last, nil
	}

	// last is unknown, use last key file
	data, err := os.ReadFile(e.path)
	if err == nil {
		// get latest from the chain
		return types.BlockNumber(binary.BigEndian.Uint32(data)), nil
	}

	block, err := cl.RPC.Chain.GetBlockLatest()
	if err != nil {
		return 0, errors.Wrap(err, "failed to get last block")
	}
	// set latest to previous block because
	return block.Block.Header.Number - 1, nil

}

type Callback func(changes []types.StorageChangeSet, meta *types.Metadata)

// Events processor receives all events starting from the given state
// and for each set of events calls callback cb
type Processor struct {
	sub substrate.Manager

	cb    Callback
	state State
}

func NewProcessor(sub substrate.Manager, cb Callback, state State) *Processor {
	return &Processor{
		sub:   sub,
		cb:    cb,
		state: state,
	}
}

func (e *Processor) eventsTo(cl *gsrpc.SubstrateAPI, meta *types.Metadata, block types.Header) error {
	//
	last, err := e.state.Get(cl)
	if err != nil {
		return errors.Wrap(err, "failed to get latest processed block")
	}

	key, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return err
	}

	start := last + 1
	end := block.Number

	// NOTE: calling QueryStorage with start/end hashes does not work
	// and always return error 'RPC call is unsafe to be called externally'
	for i := start; i <= end; i++ {
		hash, err := cl.RPC.Chain.GetBlockHash(uint64(i))
		if err != nil {
			return errors.Wrapf(err, "failed to get block hash '%d'", start)
		}

		//state.ErrUnknownBlock
		changes, err := cl.RPC.State.QueryStorageAt([]types.StorageKey{key}, hash)
		if err, ok := err.(rpc.Error); ok {
			if err.ErrorCode() == -32000 { // block is too old not in archive anymore
				log.Debug().Uint32("block", uint32(i)).Msg("block not available in archive anymore")
				continue
			}
		}

		if err != nil {
			return errors.Wrapf(err, "failed to get block with hash '%s'", hash.Hex())
		}

		e.cb(changes, meta)
	}

	return nil
}

func (e *Processor) subscribe(ctx context.Context) error {
	cl, meta, err := e.sub.Raw()
	if err != nil {
		return errors.Wrap(err, "failed to connect to chain")
	}

	defer cl.Client.Close()
	sub, err := cl.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		return errors.Wrap(err, "failed to subscribe to new blocks")
	}

	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			return err
		case block := <-sub.Chan():
			err := e.eventsTo(cl, meta, block)
			if err != nil {
				log.Error().Err(err).Msg("failed to process chain events")
				continue
			}

			if err := e.state.Set(block.Number); err != nil {
				return errors.Wrap(err, "failed to commit last block number")
			}
		}
	}
}

func (e *Processor) Start(ctx context.Context) {
	for {
		err := e.subscribe(ctx)
		if err != nil {
			<-time.After(10 * time.Second)
			continue
		}
		return
	}
}

// Wraps the event processor structure to provide a zbus compatible events stream
type EventsStream struct {
	node           uint32
	processor      *Processor
	pubCfg         chan pkg.PublicConfigEvent
	contractCancel chan pkg.ContractCancelledEvent
	contractLocked chan pkg.ContractLockedEvent

	o sync.Once
}

var (
	_ pkg.Events = (*EventsStream)(nil)
)

// NewEventsStream creates a new events stream
func NewEventsStream(sub substrate.Manager, node uint32, state string) *EventsStream {
	e := EventsStream{
		pubCfg:         make(chan pkg.PublicConfigEvent),
		contractCancel: make(chan pkg.ContractCancelledEvent),
		contractLocked: make(chan pkg.ContractLockedEvent),
	}

	processor := NewProcessor(sub, e.process, NewFileState(state))
	e.processor = processor

	return &e
}

func (e *EventsStream) process(changes []types.StorageChangeSet, meta *types.Metadata) {
	for _, set := range changes {
		for _, change := range set.Changes {
			if !change.HasStorageData {
				continue
			}

			var events substrate.EventRecords
			if err := types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, &events); err != nil {
				log.Error().Err(err).Msg("failed to decode events from tfchain")
				continue
			}

			for _, event := range events.TfgridModule_NodePublicConfigStored {
				if event.Node != types.U32(e.node) {
					continue
				}
				log.Info().Msgf("got a public config update: %+v", event.Config)
				e.pubCfg <- pkg.PublicConfigEvent{
					PublicConfig: event.Config,
				}
			}

			for _, event := range events.SmartContractModule_NodeContractCanceled {
				if event.Node != types.U32(e.node) {
					continue
				}
				log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract cancel update")
				e.contractCancel <- pkg.ContractCancelledEvent{
					Contract: uint64(event.ContractID),
					TwinId:   uint32(event.Twin),
				}
			}

			for _, event := range events.SmartContractModule_ContractGracePeriodStarted {
				if event.NodeID != types.U32(e.node) {
					continue
				}
				log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract grace period started")
				e.contractLocked <- pkg.ContractLockedEvent{
					Contract: uint64(event.ContractID),
					TwinId:   uint32(event.TwinID),
					Lock:     true,
				}
			}

			for _, event := range events.SmartContractModule_ContractGracePeriodEnded {
				if event.NodeID != types.U32(e.node) {
					continue
				}
				log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract grace period ended")
				e.contractLocked <- pkg.ContractLockedEvent{
					Contract: uint64(event.ContractID),
					TwinId:   uint32(event.TwinID),
					Lock:     false,
				}
			}
		}
	}
}

func (m *EventsStream) PublicConfigEvent(ctx context.Context) <-chan pkg.PublicConfigEvent {
	m.o.Do(func() {
		go m.processor.Start(ctx)
	})

	return m.pubCfg
}

func (m *EventsStream) ContractCancelledEvent(ctx context.Context) <-chan pkg.ContractCancelledEvent {
	m.o.Do(func() {
		go m.processor.Start(ctx)
	})

	return m.contractCancel
}

func (m *EventsStream) ContractLockedEvent(ctx context.Context) <-chan pkg.ContractLockedEvent {
	m.o.Do(func() {
		go m.processor.Start(ctx)
	})

	return m.contractLocked
}
