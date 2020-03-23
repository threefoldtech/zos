package logger

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// LoggerFile defines file logger type name
const LoggerFile = "file"

// ContainerLoggerFile write stdout/stderr to
// a defined file
type ContainerLoggerFile struct {
	filepath string
	fd       *os.File
}

// NewContainerLoggerFile open file and prepare logs writing
func NewContainerLoggerFile(filepath string) (*ContainerLoggerFile, error) {
	log.Debug().Str("filepath", filepath).Msg("initializing localfile logging")

	f, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}

	return &ContainerLoggerFile{
		filepath: filepath,
		fd:       f,
	}, nil
}

// Stdout handle a stdout single line
func (c *ContainerLoggerFile) Stdout(line string) error {
	_, err := fmt.Fprintf(c.fd, "%s\n", line)
	if err != nil {
		return err
	}

	return nil
}

// Stderr handle a stderr single line
func (c *ContainerLoggerFile) Stderr(line string) error {
	_, err := fmt.Fprintf(c.fd, "%s\n", line)
	if err != nil {
		return err
	}

	return nil
}

// CloseStdout closes stdout handler
func (c *ContainerLoggerFile) CloseStdout() {
	c.fd.Close()
}

// CloseStderr closes stderr handler
func (c *ContainerLoggerFile) CloseStderr() {
	c.fd.Close()
}
