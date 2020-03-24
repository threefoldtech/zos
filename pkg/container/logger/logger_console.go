package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// LoggerConsole defines console logger type name
const ConsoleType = "console"

// ContainerLoggerConsole does nothing else that print
// logs on console stdout/stderr, there are no config
type Console struct {
	prefix string
	target *os.File
}

// NewContainerLoggerConsole does nothing, it's here for consistancy
func NewConsole() (io.Writer, io.Writer, error) {
	log.Debug().Msg("initializing console logging")
	stdout := &Console{
		prefix: "O: ",
		target: os.Stdout,
	}

	stderr := &Console{
		prefix: "E: ",
		target: os.Stderr,
	}

	return stdout, stderr, nil
}

func (c *Console) Write(data []byte) (n int, err error) {
	fmt.Fprintf(c.target, "%s", c.prefix)

	// note: cannot use Fprintf("%s%s", c.prefix, data)
	// caller seems to expect that return amount of byte
	// matches to data length

	return c.target.Write(data)
}
