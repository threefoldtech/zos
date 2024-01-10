package utils

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisConfig struct {
	Scheme  string
	Host    string
	Options []redis.DialOption
}

func parseRedisAdd(address string) (RedisConfig, error) {
	var config RedisConfig
	u, err := url.Parse(address)
	if err != nil {
		return config, err
	}

	config.Scheme = u.Scheme

	switch config.Scheme {
	case "redis":
		config.Scheme = "tcp"
		fallthrough
	case "tcp":
		config.Host = u.Host
	case "unix":
		config.Host = u.Path
	default:
		return config, fmt.Errorf("unknown scheme '%s' expecting tcp or unix", u.Scheme)
	}

	if u.User != nil {
		config.Options = append(
			config.Options,
			redis.DialPassword(u.User.Username()),
		)
	}

	return config, nil
}
func NewRedisPool(address string, size ...uint32) (*redis.Pool, error) {
	var poolSize uint32 = 20
	if len(size) == 1 {
		poolSize = size[0]
	} else if len(size) > 1 {
		panic("invalid pool size")
	}

	config, err := parseRedisAdd(address)
	if err != nil {
		return nil, err
	}

	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(config.Scheme, config.Host, config.Options...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) > 10*time.Second {
				//only check connection if more than 10 second of inactivity
				_, err := c.Do("PING")
				return err
			}

			return nil
		},
		MaxActive:   int(poolSize),
		MaxIdle:     3,
		IdleTimeout: 1 * time.Minute,
		Wait:        true,
	}, nil
}

func NewRedisConn(address string) (redis.Conn, error) {
	config, err := parseRedisAdd(address)
	if err != nil {
		return nil, err
	}
	return redis.Dial(config.Scheme, config.Host, config.Options...)
}
