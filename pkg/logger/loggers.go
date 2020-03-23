package logger

import (
	"io"

	"github.com/containerd/containerd/cio"
)

// ContainerLoggers keeps stdout and stderr backend list
type ContainerLoggers struct {
	stdouts []io.Writer
	stderrs []io.Writer
}

// NewContainerLoggers initialize empty lists
func NewContainerLoggers() *ContainerLoggers {
	return &ContainerLoggers{
		stdouts: []io.Writer{},
		stderrs: []io.Writer{},
	}
}

// Add adds a defined backend on the list
func (c *ContainerLoggers) Add(stdout io.Writer, stderr io.Writer) {
	c.stdouts = append(c.stdouts, stdout)
	c.stderrs = append(c.stderrs, stderr)
}

// Log create the containers logs redirector
func (c *ContainerLoggers) Log() cio.Creator {
	mwo := io.MultiWriter(c.stdouts...)
	mwe := io.MultiWriter(c.stderrs...)

	return cio.NewCreator(cio.WithStreams(nil, mwo, mwe))
}
