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
	sub            *substrate.Substrate
	node           uint32
	pubCfg         chan pkg.PublicConfigEvent
	contactCancel  chan pkg.ContractCancelledEvent
	eventPersistor EventPersistor

	o sync.Once
}

func New(sub *substrate.Substrate, node uint32, eventFile string) pkg.Events {
	return &Manager{
		sub:            sub,
		node:           node,
		pubCfg:         make(chan pkg.PublicConfigEvent),
		contactCancel:  make(chan pkg.ContractCancelledEvent),
		eventPersistor: newEventPersistor(eventFile),
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

type EventPersistor struct {
	eventFile string // file to store last processed event
}

func newEventPersistor(blockFile string) EventPersistor {
	return EventPersistor{
		blockFile,
	}
}

func (p *EventPersistor) MissedEvents(sub *substrate.Substrate) ([]types.StorageChangeSet, error) {
	file, err := os.Open(p.eventFile)
	if os.IsNotExist(err) {
		return make([]types.StorageChangeSet, 0), nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "couldn't open last-event file")
	}
	firstBlockStr, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't read event file")
	}
	firstBlockHash, err := types.NewHashFromHexString(string(firstBlockStr))
	if err != nil {
		return nil, errors.Wrap(err, "couldn't convert event hex")
	}
	firstBlock, err := sub.GetBlock(firstBlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get last-event block")
	}
	lastBlock, err := sub.GetCurrentHeight()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get last chain last block")
	}
	log.Debug().
		Uint32("last_processed", uint32(firstBlock.Block.Header.Number)).
		Uint32("last_block", lastBlock).
		Msg("fetching missed events")
	_, res, err := sub.FetchEventsForBlockRange(uint32(firstBlock.Block.Header.Number), lastBlock)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't fetch missed events")
	}
	return res, nil
}

func (p *EventPersistor) Commit(last *types.StorageChangeSet) error {
	return ioutil.WriteFile(p.eventFile, []byte(last.Block.Hex()), 0644)
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

		missed, err := m.eventPersistor.MissedEvents(m.sub)
		if err != nil {
			// continuing to not block the whole event subsystem in case
			// of a blockchain reset (can it be any other reason?)
			log.Error().Err(err).Msg("failed to fetch missed events, dropping them")
		}
		for _, event := range missed {
			m.processEvent(&event, meta)
		}
		for {
			select {
			case event := <-reg.Chan():
				// events might be duplicated from missed event
				// in case a block is added after subscription
				// but before listing the missed block, not a problem?
				m.processEvent(&event, meta)
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
func (m *Manager) processEvent(event *types.StorageChangeSet, meta *types.Metadata) {
	defer func() {
		if err := m.eventPersistor.Commit(event); err != nil {
			log.Warn().Err(err).Msg("couldn't persist last processed event")
		}
	}()
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
