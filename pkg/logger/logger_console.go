package logger

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

// LoggerConsole defines console logger type name
const LoggerConsole = "console"

// ContainerLoggerConsole does nothing else that print
// logs on console stdout/stderr, there are no config
type ContainerLoggerConsole struct{}

// NewContainerLoggerConsole does nothing, it's here for consistancy
func NewContainerLoggerConsole() *ContainerLoggerConsole {
	log.Debug().Msg("initializing console logging")
	return &ContainerLoggerConsole{}
}

// Stdout handle a stdout single line
func (c *ContainerLoggerConsole) Stdout(line string) error {
	fmt.Printf("O: %s\n", line)
	return nil
}

// Stderr handle a stderr single line
func (c *ContainerLoggerConsole) Stderr(line string) error {
	fmt.Fprintf(os.Stderr, "E: %s\n", line)
	return nil
}

// CloseStdout closes stdout handler
func (c *ContainerLoggerConsole) CloseStdout() {
	// Nothing to close
}

// CloseStderr closes stderr handler
func (c *ContainerLoggerConsole) CloseStderr() {
	// Nothing to close
}
