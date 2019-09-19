package upgrade

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules/zinit"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules"
)

type hookType string

const (
	provisionModuleName = "provisiond"

	// those values must match the values
	// in the bootstrap process. (bootstrap.sh)

	// verFile the current version
	verFile = "/tmp/version"
	// bootFile the boot flist name
	bootFile = "/tmp/boot"
)

var (
	// ErrRestartNeeded is returned if upgraded requires a restart
	ErrRestartNeeded = fmt.Errorf("restart needed")

	// services that can't be updated with normal procedure
	protected = []string{"upgraded", "redis"}
)

//BootInfo reflects node boot information (flist + version)
type BootInfo struct {
	FList   string
	Version string
}

// Commit write version to version file
func (b *BootInfo) Commit(version string) error {
	b.Version = version
	return ioutil.WriteFile(verFile, []byte(version), 0400)
}

// DefaultBootInfo get boot info set by bootstrap process
func DefaultBootInfo() BootInfo {
	flist, _ := ioutil.ReadFile(bootFile)
	version, _ := ioutil.ReadFile(verFile)

	return BootInfo{
		FList:   strings.TrimSpace(string(flist)),
		Version: strings.TrimSpace(string(version)),
	}
}

// Upgrader is the component that is responsible
// to keep 0-OS up to date
type Upgrader struct {
	BootInfo
	hub     Hub
	FLister modules.Flister
	Zinit   *zinit.Client
}

// Upgrade is the method that does a full upgrade flow
// first check if a new version is available
// if yes, applies the upgrade
// on a successfully update, upgrade WILL NOT RETURN
// instead the upgraded daemon will be completely stopped
func (u *Upgrader) Upgrade() error {
	latest, err := u.hub.Hash(u.FList)
	if err != nil {
		return errors.Wrap(err, "failed to get latest hash")
	}

	if u.Version == latest {
		// nothing to do.
		return nil
	}

	return u.applyUpgrade(latest)
}

func (u Upgrader) stopMultiple(timeout time.Duration, service ...string) ([]string, error) {
	services := make(map[string]struct{})
	for _, name := range service {
		log.Info().Str("service", name).Msg("stopping service")
		if err := u.Zinit.Stop(name); err != nil {
			log.Debug().Str("service", name).Msg("service undefined")
			continue
		}

		services[name] = struct{}{}
	}

	deadline := time.After(timeout)
	var stopped []string

	for len(services) > 0 {
		for service := range services {
			status, err := u.Zinit.Status(service)
			if err != nil {
				return stopped, err
			}

			if status.Target != zinit.ServiceTargetDown {
				// it means some other entity (another client or command line)
				// has set the service back to up. I think we should immediately return
				// with an error instead.
				return stopped, fmt.Errorf("expected service target should be DOWN. found UP")
			}

			if status.State.Exited() {
				stopped = append(stopped, service)
			}
		}

		for _, stop := range stopped {
			if _, ok := services[stop]; ok {
				log.Debug().Str("service", stop).Msg("service stopped")
				delete(services, stop)
			}
		}

		if len(services) == 0 {
			break
		}

		select {
		case <-deadline:
			for service := range services {
				u.Zinit.Kill(service, syscall.SIGKILL)
			}
		case <-time.After(1 * time.Second):
		}
	}

	return stopped, nil
}

func (u *Upgrader) applyUpgrade(version string) error {
	log.Info().Str("flist", u.FList).Str("version", version).Msg("start applying upgrade")

	flistRoot, err := u.FLister.Mount(u.hub.URL(u.FList), u.hub.Storage())
	if err != nil {
		return err
	}

	defer func() {
		if err := u.FLister.Umount(flistRoot); err != nil {
			log.Error().Err(err).Msgf("fail to umount flist at %s: %v", flistRoot, err)
		}
	}()

	// once the flist is mounted we can inspect
	// it for all zinit config files.
	files, err := ioutil.ReadDir(filepath.Join(flistRoot, "etc", "zinit"))
	if err != nil {
		return errors.Wrap(err, "invalid flist. no zinit services")
	}

	var names []string
	for _, service := range files {
		name := service.Name()
		if service.IsDir() || !strings.HasSuffix(name, ".yaml") {
			continue
		}

		name = strings.TrimSuffix(name, ".yaml")
		// skip self and redis
		if isIn(name, protected) {
			continue
		}

		names = append(names, name)
	}

	stopped, err := u.stopMultiple(10*time.Second, names...)
	if err != nil {
		return errors.Wrapf(err, "failed to stop services: %+v", names)
	}

	// we do a forget so any changes of the zinit config
	// themselves get reflected once monitored again
	for _, stopped := range stopped {
		if err := u.Zinit.Forget(stopped); err != nil {
			log.Error().Err(err).Str("service", stopped).Msg("error on zinit forget")
		}
	}

	if err := copyRecursive(flistRoot, "/"); err != nil {
		return err
	}

	// start all services in the flist
	for _, name := range names {
		if err := u.Zinit.Monitor(name); err != nil {
			log.Error().Err(err).Str("service", name).Msg("error on zinit monitor")
		}

		// while we totally do not need to call start after monitor but
		// monitor won't take an action on a monitored service if it's
		// stopped (but not forgoten). So we call start just to be sure
		if err := u.Zinit.Start(name); err != nil {
			log.Error().Err(err).Str("service", name).Msg("error on zinit start")
		}
	}

	if err := u.BootInfo.Commit(version); err != nil {
		return err
	}

	return ErrRestartNeeded
}

func copyRecursive(source string, destination string, skip ...string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		if isIn(rel, skip) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		dest := filepath.Join(destination, rel)
		if info.IsDir() {
			if err := os.MkdirAll(dest, info.Mode()); err != nil {
				return err
			}
		} else {
			// regular file (or other types that we don't handle)
			if err := copyFile(dest, path); err != nil {
				return err
			}
		}

		return nil
	})
}

func isIn(target string, list []string) bool {
	for _, x := range list {
		if target == x {
			return true
		}
	}
	return false
}

func copyFile(dst, src string) error {
	log.Info().Str("source", src).Str("destination", dst).Msg("copy file")

	var (
		isNew  = false
		dstOld string
	)

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		// case where this is a new file
		// we just need to copy from flist to root
		isNew = true
	}

	if !isNew {
		dstOld = dst + ".old"
		if err := os.Rename(dst, dstOld); err != nil {
			return err
		}
	}

	fSrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fSrc.Close()

	stat, err := fSrc.Stat()
	if err != nil {
		return err
	}

	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, stat.Mode().Perm())
	if err != nil {
		return err
	}
	defer fDst.Close()

	if _, err = io.Copy(fDst, fSrc); err != nil {
		return err
	}

	if !isNew {
		return os.Remove(dstOld)
	}
	return nil
}
