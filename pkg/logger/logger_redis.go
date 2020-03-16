package logger

import (
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
)

type ContainerLoggerRedis struct {
	ContainerLogger

	Endpoint      string
	ChannelStdout string
	ChannelStderr string

	conn redis.Conn
}

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

func (c *ContainerLoggerRedis) Stdout(line string) error {
	_, err := c.conn.Do("PUBLISH", c.ChannelStdout, line)
	if err != nil {
		return err
	}

	return nil
}

func (c *ContainerLoggerRedis) Stderr(line string) error {
	_, err := c.conn.Do("PUBLISH", c.ChannelStderr, line)
	if err != nil {
		return err
	}

	return nil
}

func (c *ContainerLoggerRedis) CloseStdout() {
	c.conn.Close()
}

func (c *ContainerLoggerRedis) CloseStderr() {
	c.conn.Close()
}
