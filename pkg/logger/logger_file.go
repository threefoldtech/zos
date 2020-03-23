package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// LoggerFile defines file logger type name
const LoggerFile = "file"

// ContainerLoggerFile write stdout/stderr to files
type ContainerLoggerFile struct {
	target *os.File
}

// NewContainerLoggerFile open file and prepare logs writing
func NewContainerLoggerFile(stdout string, stderr string) (io.Writer, io.Writer, error) {
	log.Debug().Str("stdout", stdout).Str("stderr", stderr).Msg("initializing localfile logging")

	fo, err := os.Create(stdout)
	if err != nil {
		return nil, nil, err
	}

	// If stdout and stderr are the same, only one file is open
	fe := fo

	if stdout != stderr {
		fe, err = os.Create(stderr)
		if err != nil {
			return nil, nil, err
		}
	}

	fstdout := &ContainerLoggerFile{
		target: fo,
	}

	fstderr := &ContainerLoggerFile{
		target: fe,
	}

	return fstdout, fstderr, nil
}

// Write forwards write to underlaying layer
func (c *ContainerLoggerFile) Write(data []byte) (n int, err error) {
	return c.target.Write(data)
}
