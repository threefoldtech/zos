package monitord

import (
	"context"
	"os"
	"time"

	"github.com/blang/semver"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/upgrade"
)

// HostMonitor monitor host information
type hostMonitor struct {
	duration time.Duration

	watcher *fsnotify.Watcher
	version semver.Version
	boot    upgrade.Boot
}

// NewHostMonitor initialize a new host watcher
func NewHostMonitor(duration time.Duration) (pkg.HostMonitor, error) {
	if duration == 0 {
		duration = 2 * time.Second
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize fs watcher")
	}

	// the file can not exist if the system was booted from overlay
	if _, err := os.Stat(upgrade.FlistInfoFile); err == nil {
		if err := watcher.Add(upgrade.FlistInfoFile); err != nil {
			return nil, errors.Wrapf(err, "failed to watch '%s'", upgrade.FlistInfoFile)
		}
	}

	return &hostMonitor{
		watcher:  watcher,
		duration: duration,
	}, nil
}

// Version get stream of version information
func (h *hostMonitor) Version(ctx context.Context) <-chan semver.Version {
	h.version, _ = h.boot.Version()
	ch := make(chan semver.Version)
	go func() {
		defer close(ch)
		defer h.watcher.Close()

		for {
			select {
			case e := <-h.watcher.Events:
				if e.Op == fsnotify.Create || e.Op == fsnotify.Write {
					h.version, _ = h.boot.Version()
					ch <- h.version
				}
			case <-time.After(h.duration):
				ch <- h.version
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}
