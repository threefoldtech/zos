package events

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/utils"
)

const (
	maxStreamLen = 1024
	bodyTag      = "body"

	streamPublicConfig        = "stream:public-config"
	streamContractCancelled   = "stream:contract-cancelled"
	streamContractGracePeriod = "stream:contract-lock"
	streamPowerChange         = "stream:power-change"
)

type RedisStream struct {
	sub   substrate.Manager
	state string
	node  uint32
	farm  pkg.FarmID
	pool  *redis.Pool
}

func NewRedisStream(sub substrate.Manager, address string, farm pkg.FarmID, node uint32, state string) (*RedisStream, error) {
	pool, err := utils.NewRedisPool(address, 2)
	if err != nil {
		return nil, err
	}

	return &RedisStream{
		sub:   sub,
		state: state,
		node:  node,
		farm:  farm,
		pool:  pool,
	}, nil
}

func (r *RedisStream) push(con redis.Conn, queue string, event interface{}) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(event); err != nil {
		return errors.Wrap(err, "failed to encode event")
	}

	_, err := con.Do("XADD",
		queue,
		"MAXLEN", "~", maxStreamLen,
		"*",
		bodyTag, buffer.Bytes())

	return err
}

func (r *RedisStream) process(events *substrate.EventRecords) {
	con := r.pool.Get()
	defer con.Close()

	for _, event := range events.TfgridModule_NodePublicConfigStored {
		if event.Node != types.U32(r.node) {
			continue
		}
		log.Info().Msgf("got a public config update: %+v", event.Config)

		if err := r.push(con, streamPublicConfig, pkg.PublicConfigEvent{
			PublicConfig: event.Config,
		}); err != nil {
			log.Error().Err(err).Msg("failed to push event")
		}
	}

	for _, event := range events.SmartContractModule_DeploymentCanceled {
		if event.NodeID != types.U32(r.node) {
			continue
		}
		log.Info().Uint64("deployment", uint64(event.DeploymentID)).Msg("got contract cancel update")
		if err := r.push(con, streamContractCancelled, pkg.DeploymentCancelledEvent{
			Deployment: gridtypes.DeploymentID(event.DeploymentID),
			TwinId:     uint32(event.TwinID),
		}); err != nil {
			log.Error().Err(err).Msg("failed to push event")
		}
	}

	for _, event := range events.SmartContractModule_ContractGracePeriodStarted {
		if event.NodeID != types.U32(r.node) {
			continue
		}
		log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract grace period started")
		if err := r.push(con, streamContractGracePeriod, pkg.ContractLockedEvent{
			Contract: gridtypes.ContractID(event.ContractID),
			TwinId:   uint32(event.TwinID),
			Lock:     true,
		}); err != nil {
			log.Error().Err(err).Msg("failed to push event")
		}
	}

	for _, event := range events.SmartContractModule_ContractGracePeriodEnded {
		if event.NodeID != types.U32(r.node) {
			continue
		}
		log.Info().Uint64("contract", uint64(event.ContractID)).Msg("got contract grace period ended")
		if err := r.push(con, streamContractGracePeriod, pkg.ContractLockedEvent{
			Contract: gridtypes.ContractID(event.ContractID),
			TwinId:   uint32(event.TwinID),
			Lock:     false,
		}); err != nil {
			log.Error().Err(err).Msg("failed to push event")
		}
	}

	for _, event := range events.TfgridModule_PowerTargetChanged {
		if event.FarmID != types.U32(r.farm) {
			continue
		}

		log.Info().Uint64("node", uint64(event.NodeID)).Uint64("farm", uint64(event.FarmID)).Msg("got power change event")
		if err := r.push(con, streamPowerChange, pkg.PowerChangeEvent{
			FarmID: pkg.FarmID(event.FarmID),
			NodeID: uint32(event.NodeID),
			Target: event.PowerTarget,
		}); err != nil {
			log.Error().Err(err).Msg("failed to push event")
		}
	}

}

func (r *RedisStream) Start(ctx context.Context) {
	ps := NewProcessor(r.sub, r.process, NewFileState(r.state))
	ps.Start(ctx)
}

type RedisConsumer struct {
	id   string
	pool *redis.Pool
}

func NewConsumer(address, id string) (*RedisConsumer, error) {
	pool, err := utils.NewRedisPool(address)
	if err != nil {
		return nil, err
	}

	return &RedisConsumer{
		id:   id,
		pool: pool,
	}, nil
}

func (r *RedisConsumer) ensureGroup(con redis.Conn, stream string) (string, error) {
	group := fmt.Sprintf("group:%s:%s", streamPublicConfig, r.id)
	_, err := con.Do("XGROUP",
		"CREATE", stream,
		group,
		0, "MKSTREAM")
	return group, err
}

func (r *RedisConsumer) pop(con redis.Conn, group, stream string) ([]payload, error) {
	// check if we have pending messages
	streams, err := intoPayloads(con.Do(
		"XREADGROUP",
		"GROUP", group, r.id,
		"COUNT", 128,
		"BLOCK", 0,
		"STREAMS", stream,
		0))

	if err != nil {
		return nil, err
	}

	messages := streams[stream]
	if len(messages) > 0 {
		return messages, nil
	}

	// otherwise we just wait for new messages
	streams, err = intoPayloads(con.Do(
		"XREADGROUP",
		"GROUP", group, r.id,
		"COUNT", 1,
		"BLOCK", 3000,
		"STREAMS", stream,
		">"))

	if err != nil {
		return nil, err
	}

	messages = streams[stream]
	return messages, nil
}

func (r *RedisConsumer) ack(ctx context.Context, con redis.Conn, group, stream, id string) error {
	_, err := con.Do("XACK", stream, group, id)
	return err
}

func (r *RedisConsumer) PublicConfig(ctx context.Context) (<-chan pkg.PublicConfigEvent, error) {
	con := r.pool.Get()
	ch := make(chan pkg.PublicConfigEvent)

	const stream = streamPublicConfig
	group, err := r.ensureGroup(con, stream)

	if err != nil && !isBusyGroup(err) {
		return nil, err
	}

	logger := log.With().Str("stream", stream).Logger()
	go func() {
		defer con.Close()

		for {
			messages, err := r.pop(con, group, stream)
			if err != nil {
				logger.Error().Err(err).Msg("failed to get events from")
			}

			for _, message := range messages {
				var event pkg.PublicConfigEvent
				if err := message.Decode(&event); err == nil {
					select {
					case ch <- event:
					case <-ctx.Done():
						return
					}
				} else if err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to handle message")
				}

				if err := r.ack(ctx, con, group, stream, message.ID); err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to ack message")
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return ch, nil
}

func (r *RedisConsumer) ContractCancelled(ctx context.Context) (<-chan pkg.DeploymentCancelledEvent, error) {
	con := r.pool.Get()
	ch := make(chan pkg.DeploymentCancelledEvent)

	const stream = streamContractCancelled
	group, err := r.ensureGroup(con, stream)

	if err != nil && !isBusyGroup(err) {
		return nil, err
	}

	logger := log.With().Str("stream", stream).Logger()
	go func() {
		defer con.Close()

		for {
			messages, err := r.pop(con, group, stream)
			if err != nil {
				logger.Error().Err(err).Msg("failed to get events from")
			}

			for _, message := range messages {
				var event pkg.DeploymentCancelledEvent
				if err := message.Decode(&event); err == nil {
					select {
					case ch <- event:
					case <-ctx.Done():
						return
					}
				} else if err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to handle message")
				}

				if err := r.ack(ctx, con, group, stream, message.ID); err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to ack message")
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return ch, nil
}

func (r *RedisConsumer) ContractLocked(ctx context.Context) (<-chan pkg.ContractLockedEvent, error) {
	con := r.pool.Get()
	ch := make(chan pkg.ContractLockedEvent)

	const stream = streamContractGracePeriod
	group, err := r.ensureGroup(con, stream)

	if err != nil && !isBusyGroup(err) {
		return nil, err
	}

	logger := log.With().Str("stream", stream).Logger()
	go func() {
		defer con.Close()

		for {
			messages, err := r.pop(con, group, stream)
			if err != nil {
				logger.Error().Err(err).Msg("failed to get events from")
			}

			for _, message := range messages {
				var event pkg.ContractLockedEvent
				if err := message.Decode(&event); err == nil {
					select {
					case <-ctx.Done():
						return
					case ch <- event:
					}
				} else if err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to handle message")
				}

				if err := r.ack(ctx, con, group, stream, message.ID); err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to ack message")
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return ch, nil
}

func (r *RedisConsumer) PowerChange(ctx context.Context) (<-chan pkg.PowerChangeEvent, error) {
	con := r.pool.Get()
	ch := make(chan pkg.PowerChangeEvent)

	const stream = streamPowerChange
	group, err := r.ensureGroup(con, stream)

	if err != nil && !isBusyGroup(err) {
		return nil, err
	}

	logger := log.With().Str("stream", stream).Logger()
	go func() {
		defer con.Close()

		for {
			messages, err := r.pop(con, group, stream)
			if err != nil {
				logger.Error().Err(err).Msg("failed to get events from")
			}

			for _, message := range messages {
				var event pkg.PowerChangeEvent
				if err := message.Decode(&event); err == nil {
					select {
					case <-ctx.Done():
						return
					case ch <- event:
					}
				} else if err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to handle message")
				}

				if err := r.ack(ctx, con, group, stream, message.ID); err != nil {
					logger.Error().Err(err).Str("id", message.ID).Msg("failed to ack message")
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return ch, nil
}

type payload struct {
	ID   string
	Tags map[string][]byte
}

func (m *payload) Decode(obj interface{}) error {
	body, ok := m.Tags[bodyTag]
	if !ok {
		return fmt.Errorf("message has no body")
	}

	dec := gob.NewDecoder(bytes.NewBuffer(body))
	if err := dec.Decode(obj); err != nil {
		return err
	}

	return nil
}

func intoPayloads(result interface{}, err error) (map[string][]payload, error) {
	if err != nil {
		return nil, err
	}

	output := make(map[string][]payload)
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("invalid type: %v", p)
		}
	}()

	streams := result.([]interface{})
	for _, streamI := range streams {
		stream := streamI.([]interface{})
		key := string(stream[0].([]byte))
		elements := stream[1].([]interface{})
		var messages []payload
		for _, elementI := range elements {
			element := elementI.([]interface{})
			id := string(element[0].([]byte))
			tags := element[1].([]interface{})
			message := payload{
				ID:   id,
				Tags: make(map[string][]byte),
			}
			for i := 0; i < len(tags); i += 2 {
				key := string(tags[0].([]byte))
				body := tags[1].([]byte)
				message.Tags[key] = body
			}

			messages = append(messages, message)
		}

		output[key] = messages
	}

	return output, nil
}

func isBusyGroup(err error) bool {
	if err == nil {
		return false
	} else if strings.HasPrefix(err.Error(), "BUSYGROUP") {
		return true
	}

	return false
}
