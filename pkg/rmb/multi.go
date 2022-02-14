package rmb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ResponseMetadata struct {
	err  error
	twin uint32
}

func NewResponseMetadata() ResponseMetadata {
	return ResponseMetadata{nil, 0}
}
func (r *ResponseMetadata) SetError(err error) {
	r.err = err
}
func (r *ResponseMetadata) SetTwin(twin uint32) {
	r.twin = twin
}
func (r *ResponseMetadata) Twin() uint32 {
	return r.twin
}
func (r *ResponseMetadata) Error() error {
	return r.err
}

type MultiRMB struct {
	pool *redis.Pool
}

func DefaultMultiRMB() (MultiDestinationClient, error) {
	return NewMultiRMBClient(DefaultAddress)
}

func NewMultiRMBClient(address string, poolSize ...uint32) (MultiDestinationClient, error) {

	if len(address) == 0 {
		address = DefaultAddress
	}

	pool, err := newRedisPool(address, poolSize...)
	if err != nil {
		return nil, err
	}

	return &MultiRMB{
		pool: pool,
	}, nil
}

func (c *MultiRMB) Call(ctx context.Context, twins []uint32, fn string, data interface{}, constructor func() Response) (chan Response, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize request data")
	}

	queue := uuid.NewString()
	msg := Message{
		Version:    1,
		Expiration: 3600,
		Command:    fn,
		TwinDest:   twins,
		Data:       base64.StdEncoding.EncodeToString(bytes),
		Retqueue:   queue,
	}

	bytes, err = json.Marshal(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize message")
	}
	con := c.pool.Get()

	_, err = con.Do("RPUSH", systemLocalBus, bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to push message to local twin")
	}
	result := make(chan Response)
	go func() {
		defer close(result)
		defer con.Close()
		remaining := make(map[uint32]struct{})
		for _, twin := range twins {
			remaining[twin] = struct{}{}
		}
		// now wait for response.
		for {
			select {
			case <-ctx.Done():
				for twin := range remaining {
					response := constructor()
					response.SetTwin(twin)
					response.SetError(ctx.Err())
					result <- response
				}
				return
			default:
			}

			slice, err := redis.ByteSlices(con.Do("BLPOP", queue, 5))
			if err != nil && err != redis.ErrNil {
				log.Error().Err(err).Msg("unexpected error during waiting for the response")
				break
			}

			if err == redis.ErrNil || slice == nil {
				//timeout, just try again immediately
				continue
			}
			response := constructor()
			// found a response
			bytes = slice[1]

			twin, data, err := processResponse(bytes)
			response.SetTwin(twin)
			if err != nil {
				response.SetError(err)
			} else if data != nil {
				if err := response.SetResponse(data); err != nil {
					response.SetError(errors.Wrap(err, "couldn't decode response"))
				}
			}
			result <- response
			delete(remaining, twin)
			log.Debug().Uint32("twin", twin).Int("remaining_count", len(remaining)).Str("remaining", fmt.Sprintf("%v", remaining)).Msg("sending response")
			if len(remaining) == 0 {
				break
			}
		}
	}()
	return result, nil

}

func processResponse(bytes []byte) (uint32, []byte, error) {
	var msg Message
	// we have a response, so load or fail
	if err := json.Unmarshal(bytes, &msg); err != nil {
		return msg.TwinSrc, nil, errors.Wrap(err, "failed to load response message")
	}

	// errorred ?
	if len(msg.Err) != 0 {
		return msg.TwinSrc, nil, errors.New(msg.Err)
	}

	if len(msg.Data) == 0 {
		return msg.TwinSrc, nil, fmt.Errorf("no response body was returned")
	}

	bytes, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		return msg.TwinSrc, nil, errors.Wrap(err, "invalid data body encoding")
	}

	return msg.TwinSrc, bytes, nil
}
