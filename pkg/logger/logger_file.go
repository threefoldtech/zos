package logger

import (
	"github.com/rs/zerolog/log"
	"os"
)

type ContainerLoggerFile struct {
	ContainerLogger

	Filepath string
	fd       *os.File
}

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

func (c *ContainerLoggerFile) Stdout(line string) error {
	_, err := c.fd.WriteString(line + "\n")
	if err != nil {
		return err
	}

	return nil
}

func (c *ContainerLoggerFile) Stderr(line string) error {
	_, err := c.fd.WriteString(line + "\n")
	if err != nil {
		return err
	}

	return nil
}

func (c *ContainerLoggerFile) CloseStdout() {
	c.fd.Close()
}

func (c *ContainerLoggerFile) CloseStderr() {
	c.fd.Close()
}
