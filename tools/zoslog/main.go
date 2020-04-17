package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/containerd/containerd/runtime/v2/logging"
	"github.com/threefoldtech/zos/pkg/container/logger"
)

var (
	ROOT_DIR   = "/var/cache/modules/contd/"
	LOGS_DIR   = "logs"
	CONFIG_DIR = "config"
)

func main() {
	logging.Run(runlog)
}

func runlog(ctx context.Context, config *logging.Config, ready func() error) error {
	// initializing container logger
	cfgfile := path.Join(ROOT_DIR, CONFIG_DIR, config.Namespace, fmt.Sprintf("%s-logs.json", config.ID))

	// load config saved by contd
	logs, err := logger.Deserialize(cfgfile)
	if err != nil {
		return err
	}

	// initializing logs endpoints
	loggers := logger.NewLoggers()

	// create local default logs directory
	local := path.Join(ROOT_DIR, LOGS_DIR, config.Namespace)
	if err = os.MkdirAll(local, 0755); err != nil {
		return err
	}

	// hardcode local logfile
	filepath := path.Join(local, fmt.Sprintf("%s.log", config.ID))
	fileout, fileerr, err := logger.NewFile(filepath, filepath)
	if err != nil {
		return err
	}

	loggers.Add(fileout, fileerr)

	// set user defined endpoint logging
	for _, l := range logs {
		switch l.Type {
		case logger.RedisType:
			lo, le, err := logger.NewRedis(l.Data.Stdout, l.Data.Stderr)

			if err != nil {
				log.Error().Err(err).Msg("redis logger")
				continue
			}

			loggers.Add(lo, le)

		default:
			log.Error().Str("type", l.Type).Msg("invalid logging type requested")
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// forward both stdout and stderr to our backends
	go copy(&wg, config.Stdout, loggers.Stdouts())
	go copy(&wg, config.Stderr, loggers.Stderrs())

	// signal that we are ready and setup for the container to be started
	if err := ready(); err != nil {
		return err
	}

	wg.Wait()
	return nil
}

func copy(wg *sync.WaitGroup, r io.Reader, writers []io.Writer) {
	defer wg.Done()

	s := bufio.NewScanner(r)
	for s.Scan() {
		if s.Err() != nil {
			return
		}

		// send each line to each backend
		for _, x := range writers {
			_, err := x.Write([]byte(s.Text() + "\n"))
			log.Error().Err(err).Msg("forwarding backend")
		}
	}
}
