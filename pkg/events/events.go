package events

import (
	"context"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
)

type Manager struct {
	sub           *substrate.Substrate
	node          uint32
	pubCfg        chan pkg.PublicConfigEvent
	contactCancel chan pkg.ContractCancelledEvent

	o sync.Once
}

func New(sub *substrate.Substrate, node uint32) pkg.Events {
	return &Manager{
		sub:           sub,
		node:          node,
		pubCfg:        make(chan pkg.PublicConfigEvent),
		contactCancel: make(chan pkg.ContractCancelledEvent),
	}
}

func (m *Manager) start(ctx context.Context) {
	log.Info().Msg("start listening to chain events")
	for {
		if err := m.listen(ctx); err != nil {
			log.Error().Err(err).Msg("listening to events failed, retry in 10 seconds")
			<-time.After(10 * time.Second)
		}
	}
}

type PersistentEventChannel struct {
	ch        chan types.StorageChangeSet
	regCh     <-chan types.StorageChangeSet
	sub       *substrate.Substrate
	blockFile string // file to store last processed block
}

func newPersistentEventChannel(ch <-chan types.StorageChangeSet, sub *substrate.Substrate, blockFile string) (PersistentEventChannel, error) {
	res := PersistentEventChannel{
		nil,
		ch,
		sub,
		blockFile,
	}
	missed, err := res.missedEvents(sub)
	if err != nil {
		return res, err
	}
	wrapper := make(chan types.StorageChangeSet, len(missed))
	for _, e := range missed {
		wrapper <- e
	}
	go func() {
		defer close(wrapper)
		for {
			// TODO: dedup
			v, ok := <-ch

			if ok {
				wrapper <- v
			} else {
				log.Debug().Msg("parent channel closed")
				return
			}
		}
	}()
	res.ch = wrapper
	return res, nil
}

func (p *PersistentEventChannel) missedEvents(sub *substrate.Substrate) ([]types.StorageChangeSet, error) {
	file, err := os.Open(p.blockFile)
	if os.IsNotExist(err) {
		return make([]types.StorageChangeSet, 0), nil
	}
	if err != nil {
		return nil, err
	}
	firstBlockStr, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	firstBlockHash, err := types.NewHashFromHexString(string(firstBlockStr))
	if err != nil {
		return nil, err
	}
	firstBlock, err := sub.GetBlock(firstBlockHash)
	if err != nil {
		return nil, err
	}
	lastBlock, err := sub.GetCurrentHeight()
	if err != nil {
		return nil, err
	}
	_, res, err := sub.FetchEventsForBlockRange(uint32(firstBlock.Block.Header.Number), lastBlock)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (p *PersistentEventChannel) Chan() chan types.StorageChangeSet {
	return p.ch
}

func (p *PersistentEventChannel) Commit(last *types.StorageChangeSet) error {
	return ioutil.WriteFile(p.blockFile, []byte(last.Block.Hex()), 0644)
}

// Start subscribing and producing events
func (m *Manager) listen(ctx context.Context) error {
reconnect:
	for {
		subClient, meta, err := m.sub.GetClient()
		if err != nil {
			return errors.Wrap(err, "failed to get client to substrate")
		}

		m.pubCfg <- pkg.PublicConfigEvent{Kind: pkg.EventSubscribed}
		m.contactCancel <- pkg.ContractCancelledEvent{Kind: pkg.EventSubscribed}

		// Subscribe to system events via storage
		key, err := types.CreateStorageKey(meta, "System", "Events", nil)
		if err != nil {
			return errors.Wrap(err, "failed to get storage key for events")
		}

		reg, err := subClient.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
		if err != nil {
			return errors.Wrap(err, "failed to subscribe to events")
		}
		wrapper, err := newPersistentEventChannel(reg.Chan(), m.sub, "somewhere")
		if err != nil {
			return errors.Wrap(err, "failed to prepare events channel")
		}
		for {
			select {
			case event := <-wrapper.Chan():
				//m.sub.GetBlock(block types.Hash)
				for _, change := range event.Changes {
					if !change.HasStorageData {
						continue
					}

					var events substrate.EventRecords
					if err := types.EventRecordsRaw(change.StorageData).DecodeEventRecords(meta, &events); err != nil {
						log.Error().Err(err).Msg("failed to decode events from tfchain")
						continue
					}

					for _, e := range events.TfgridModule_NodePublicConfigStored {
						if e.Node != types.U32(m.node) {
							continue
						}
						log.Info().Msgf("got a public config update: %+v", e.Config)
						m.pubCfg <- pkg.PublicConfigEvent{
							Kind:         pkg.EventReceived,
							PublicConfig: e.Config,
						}
					}

					for _, e := range events.SmartContractModule_NodeContractCanceled {
						if e.Node != types.U32(m.node) {
							continue
						}
						log.Info().Uint64("contract", uint64(e.ContractID)).Msg("got contract cancel update")
						m.contactCancel <- pkg.ContractCancelledEvent{
							Kind:     pkg.EventReceived,
							Contract: uint64(e.ContractID),
							TwinId:   uint32(e.Twin),
						}
					}
				}
				if err := wrapper.Commit(&event); err != nil {
					log.Error().Err(err).Msg("failed to commit event")
				}
			case err := <-reg.Err():
				// need a reconnect
				log.Warn().Err(err).Msg("subscription to events stopped, reconnecting")
				reg.Unsubscribe()
				continue reconnect
			case <-ctx.Done():
				reg.Unsubscribe()
				return ctx.Err()
			}
		}
	}
}

func (m *Manager) PublicConfigEvent(ctx context.Context) <-chan pkg.PublicConfigEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.pubCfg
}

func (m *Manager) ContractCancelledEvent(ctx context.Context) <-chan pkg.ContractCancelledEvent {
	m.o.Do(func() {
		go m.start(ctx)
	})

	return m.contactCancel
}
