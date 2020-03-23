package logger

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
	"io"
)

// LoggerRedis defines redis logger type name
const LoggerRedis = "redis"

// ContainerLoggerRedis send stdout/stderr to a
// Redis PubSub channel
type ContainerLoggerRedis struct {
	endpoint string
	channel  string
	conn     redis.Conn
}

// NewContainerLoggerRedis create new redis backend and initialize connection
func NewContainerLoggerRedis(endpoint string, stdout string, stderr string) (io.WriteCloser, io.WriteCloser, error) {
	log.Debug().Str("endpoint", endpoint).Msg("initializing redis logging")
	log.Debug().Str("stdout", stdout).Str("stderr", stderr).Msg("redis channels")

	cout, err := redis.DialURL(endpoint)
	if err != nil {
		return nil, nil, err
	}

	cerr, err := redis.DialURL(endpoint)
	if err != nil {
		cout.Close()
		return nil, nil, err
	}

	rstdout := &ContainerLoggerRedis{
		endpoint: endpoint,
		channel:  stdout,
		conn:     cout,
	}

	rstderr := &ContainerLoggerRedis{
		endpoint: endpoint,
		channel:  stderr,
		conn:     cerr,
	}

	return rstdout, rstderr, nil
}

// Write will write to the channel
func (c *ContainerLoggerRedis) Write(data []byte) (int, error) {
	_, err := c.conn.Do("PUBLISH", c.channel, data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

// Close closes redis connection
func (c *ContainerLoggerRedis) Close() error {
	log.Debug().Str("channel", c.channel).Msg("closing redis backend")
	c.conn.Close()
	return nil
}
