package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/containerd/containerd/runtime/v2/logging"
	"github.com/threefoldtech/zos/pkg/container/logger"
)

var (
	// RootDir is the contd module root directory
	RootDir = "/var/cache/modules/contd/"

	// LogsDir points to logs directory
	LogsDir = "logs"

	// ConfigDir points to config (logs settings) directory
	ConfigDir = "config"
)

func main() {
	logging.Run(runlog)
}

func addlocal(config *logging.Config, loggers *logger.Loggers) error {
	// create local default logs directory
	local := path.Join(RootDir, LogsDir, config.Namespace)
	if err := os.MkdirAll(local, 0755); err != nil {
		return err
	}

	// hardcode local logfile
	filepath := filepath.Join(local, fmt.Sprintf("%s.log", config.ID))
	fileout, fileerr, err := logger.NewFile(filepath, filepath)
	if err != nil {
		return err
	}

	loggers.Add(fileout, fileerr)

	return nil
}

func runlog(ctx context.Context, config *logging.Config, ready func() error) error {
	// initializing container logger
	cfgfile := filepath.Join(RootDir, ConfigDir, config.Namespace, fmt.Sprintf("%s-logs.json", config.ID))

	// load config saved by contd
	logs, err := logger.Deserialize(cfgfile)
	if err != nil {
		return err
	}

	// initializing logs endpoints
	loggers := logger.NewLoggers()

	// add default local log file (skipped by default)
	// addlocal(config, loggers)

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
