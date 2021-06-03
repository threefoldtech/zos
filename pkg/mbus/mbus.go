package mbus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type Message struct {
	Version    int    `json:"version"`
	Id         string `json:"id"`
	Command    string `json:"command"`
	Expiration int    `json:"expiration"`
	Retry      int    `json:"retry"`
	Data       string `json:"data"`
	Twin_src   []int  `json:"twin_src"`
	Twin_dest  []int  `json:"twin_dest"`
	Retqueue   string `json:"retqueue"`
	Schema     string `json:"schema"`
	Epoch      int    `json:"epoch"`
	Err        string `json:"err"`
}

type Messagebus struct {
	Context context.Context
	pool    *redis.Pool
}

func New(port uint16, context context.Context) (*Messagebus, error) {
	addr := fmt.Sprintf("tcp://[::]:%d", port)
	pool, err := newRedisPool(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", addr)
	}

	return &Messagebus{
		pool:    pool,
		Context: context,
	}, nil
}

func (m *Messagebus) StreamMessages(ctx context.Context, messageChan chan Message) error {
	con := m.pool.Get()

	for {
		if ctx.Err() != nil {
			return nil
		}

		data, err := redis.ByteSlices(con.Do("BLPOP", "msgbus.system.local", 0))
		if err != nil {
			log.Err(err).Msg("failed to read from system local messagebus")
			return err
		}

		candidate := data[1]

		var m Message
		err = json.Unmarshal(candidate, &m)
		if err != nil {
			log.Err(err).Msg("failed to unmarshal message")
			return err
		}

		messageChan <- m
	}
}

func (m *Messagebus) PushMessage(ctx context.Context, message Message) error {
	con := m.pool.Get()

	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = con.Do("RPUSH", "msgbus.system.local", bytes)
	if err != nil {
		log.Err(err).Msg("failed to push to local messagebus")
		return err
	}

	return nil
}

func newRedisPool(address string) (*redis.Pool, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	var host string
	switch u.Scheme {
	case "tcp":
		host = u.Host
	case "unix":
		host = u.Path
	default:
		return nil, fmt.Errorf("unknown scheme '%s' expecting tcp or unix", u.Scheme)
	}
	var opts []redis.DialOption

	if u.User != nil {
		opts = append(
			opts,
			redis.DialPassword(u.User.Username()),
		)
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(u.Scheme, host, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > 10*time.Second {
				//only check connection if more than 10 second of inactivity
				_, err := c.Do("PING")
				return err
			}

			return nil
		},
		MaxActive:   3,
		MaxIdle:     3,
		IdleTimeout: 1 * time.Minute,
		Wait:        true,
	}, nil
}
