package logger

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
)

// ContainerLoggerRedis send stdout/stderr to a
// Redis PubSub channel
type ContainerLoggerRedis struct {
	ContainerLogger

	Endpoint      string
	ChannelStdout string
	ChannelStderr string

	conn redis.Conn
}

// NewContainerLoggerRedis create new redis backend and initialize connection
func NewContainerLoggerRedis(endpoint string, channel string) (*ContainerLoggerRedis, error) {
	log.Debug().Str("endpoint", endpoint).Str("channel", channel).Msg("initializing redis logging")

	c, err := redis.DialURL(endpoint)
	if err != nil {
		return &ContainerLoggerRedis{}, err
	}

	return &ContainerLoggerRedis{
		Endpoint:      endpoint,
		ChannelStdout: channel,
		ChannelStderr: channel,
		conn:          c,
	}, nil
}

// Stdout handle a stdout single line
func (c *ContainerLoggerRedis) Stdout(line string) error {
	_, err := c.conn.Do("PUBLISH", c.ChannelStdout, line)
	if err != nil {
		return err
	}

	return nil
}

// Stdout handle a stderr single line
func (c *ContainerLoggerRedis) Stderr(line string) error {
	_, err := c.conn.Do("PUBLISH", c.ChannelStderr, line)
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
