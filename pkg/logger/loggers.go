package logger

import (
	"bufio"
	"context"
	"sync"

	"github.com/containerd/containerd/cio"
	"github.com/rs/zerolog/log"
)

// ContainerLoggers support containerd custom logs redirection
type ContainerLoggers struct {
	// Internal containerd logger link
	direct *cio.DirectIO
	wg     sync.WaitGroup

	// List of backends
	loggers []ContainerLogger
}

// NewContainerLoggers initialize struct for containerd support
func NewContainerLoggers(ctx context.Context) (*ContainerLoggers, error) {
	fifos, err := cio.NewFIFOSetInDir("", "", false)
	if err != nil {
		return nil, err
	}

	direct, err := cio.NewDirectIO(ctx, fifos)
	if err != nil {
		return nil, err
	}

	return &ContainerLoggers{
		direct:  direct,
		loggers: []ContainerLogger{},
	}, nil
}

// Add adds a defined backend on the list
func (c *ContainerLoggers) Add(backend ContainerLogger) {
	c.loggers = append(c.loggers, backend)
}

// Log is the function to be passed to container to handle logs redirection
func (c *ContainerLoggers) Log(id string) (cio.IO, error) {
	c.wg.Add(2)

	go func() {
		defer c.wg.Done()

		scanner := bufio.NewScanner(c.direct.Stdout)

		for scanner.Scan() {
			for _, logger := range c.loggers {
				logger.Stdout(scanner.Text())
			}
		}

		if err := scanner.Err(); err != nil {
			log.Error().Err(err).Msg("stdout logging")
		}
	}()

	go func() {
		defer c.wg.Done()

		scanner := bufio.NewScanner(c.direct.Stderr)

		for scanner.Scan() {
			for _, logger := range c.loggers {
				logger.Stderr(scanner.Text())
			}
		}

		if err := scanner.Err(); err != nil {
			log.Error().Err(err).Msg("stderr logging")
		}
	}()

	go func() {
		// wait for logs to ends
		// then cleanup
		c.wg.Wait()

		// closing backends
		for _, logger := range c.loggers {
			logger.CloseStdout()
			logger.CloseStderr()
		}

		// closing containerd logs
		c.direct.Close()
	}()

	return c.direct, nil
}
