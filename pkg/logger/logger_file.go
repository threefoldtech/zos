package logger

import (
	"github.com/rs/zerolog/log"
	"os"
)

// ContainerLoggerFile write stdout/stderr to
// a defined file
type ContainerLoggerFile struct {
	ContainerLogger

	Filepath string
	fd       *os.File
}

// NewContainerLoggerFile open file and prepare logs writing
func NewContainerLoggerFile(filepath string) (*ContainerLoggerFile, error) {
	log.Debug().Str("filepath", filepath).Msg("initializing localfile logging")

	f, err := os.Create(filepath)
	if err != nil {
		return &ContainerLoggerFile{}, err
	}

	return &ContainerLoggerFile{
		Filepath: filepath,
		fd:       f,
	}, nil
}

// Stdout handle a stdout single line
func (c *ContainerLoggerFile) Stdout(line string) error {
	_, err := c.fd.WriteString(line + "\n")
	if err != nil {
		return err
	}

	return nil
}

// Stderr handle a stderr single line
func (c *ContainerLoggerFile) Stderr(line string) error {
	_, err := c.fd.WriteString(line + "\n")
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
