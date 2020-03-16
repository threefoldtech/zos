package logger

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

type ContainerLoggerConsole struct {
	ContainerLogger
}

func NewContainerLoggerConsole() *ContainerLoggerConsole {
	log.Debug().Msg("initializing console logging")
	return &ContainerLoggerConsole{}
}

func (c *ContainerLoggerConsole) Stdout(line string) error {
	fmt.Printf("O: %s\n", line)
	return nil
}

func (c *ContainerLoggerConsole) Stderr(line string) error {
	fmt.Fprintf(os.Stderr, "E: %s\n", line)
	return nil
}

func (c *ContainerLoggerConsole) CloseStdout() {
	// Nothing to close
}

func (c *ContainerLoggerConsole) CloseStderr() {
	// Nothing to close
}
