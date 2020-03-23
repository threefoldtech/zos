package logger

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
)

// LoggerRedis defines redis logger type name
const LoggerRedis = "redis"

// ContainerLoggerRedis send stdout/stderr to a
// Redis PubSub channel
type ContainerLoggerRedis struct {
	endpoint      string
	channelStdout string
	channelStderr string
	conn          redis.Conn
}

// NewContainerLoggerRedis create new redis backend and initialize connection
func NewContainerLoggerRedis(endpoint string, channel string) (*ContainerLoggerRedis, error) {
	log.Debug().Str("endpoint", endpoint).Str("channel", channel).Msg("initializing redis logging")

	c, err := redis.DialURL(endpoint)
	if err != nil {
		return nil, err
	}

	return &ContainerLoggerRedis{
		endpoint:      endpoint,
		channelStdout: channel,
		channelStderr: channel,
		conn:          c,
	}, nil
}

// Stdout handle a stdout single line
func (c *ContainerLoggerRedis) Stdout(line string) error {
	_, err := c.conn.Do("PUBLISH", c.channelStdout, line)
	if err != nil {
		return err
	}

	return nil
}

// Stderr handle a stderr single line
func (c *ContainerLoggerRedis) Stderr(line string) error {
	_, err := c.conn.Do("PUBLISH", c.channelStderr, line)
	if err != nil {
		return err
	}

	return nil
}

// CloseStdout closes stdout handler
func (c *ContainerLoggerRedis) CloseStdout() {
	c.conn.Close()
}

// CloseStderr closes stderr handler
func (c *ContainerLoggerRedis) CloseStderr() {
	c.conn.Close()
}
