package monitord

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/blang/semver"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/upgrade"
	"github.com/vishvananda/netlink"
)

// HostMonitor monitor host information
type hostMonitor struct {
	duration time.Duration

	// version related
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

func (h *hostMonitor) IPs(ctx context.Context) <-chan pkg.NetlinkAddresses {
	updates := make(chan netlink.AddrUpdate)
	if err := netlink.AddrSubscribe(updates, ctx.Done()); err != nil {
		log.Fatal().Err(err).Msg("failed to listen to netlink address updates")
	}

	link, err := netlink.LinkByName(network.DefaultBridge)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not find the '%s' bridge", network.DefaultBridge)
	}

	get := func() pkg.NetlinkAddresses {
		var result pkg.NetlinkAddresses
		values, _ := netlink.AddrList(link, netlink.FAMILY_ALL)
		for _, value := range values {
			result = append(result, pkg.NetlinkAddress(value))
		}

		return result
	}

	addresses := get()

	ch := make(chan pkg.NetlinkAddresses)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case update := <-updates:
				if update.LinkIndex != link.Attrs().Index {
					continue
				}

				addresses = get()
			case <-time.After(h.duration):
				ch <- addresses
			}
		}
	}()

	return ch

}

func (h *hostMonitor) Uptime(ctx context.Context) <-chan time.Duration {
	ch := make(chan time.Duration)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(h.duration):
				data, err := ioutil.ReadFile("/proc/uptime")
				if err != nil {
					log.Error().Err(err).Msg("failed to read data from /proc/uptime")
					continue
				}
				var uptime float64
				if _, err := fmt.Sscanf(string(data), "%f", &uptime); err != nil {
					log.Error().Err(err).Msg("failed to parse uptime data")
					continue
				}

				ch <- time.Duration(uptime * float64(time.Second))
			}
		}
	}()

	return ch
}
