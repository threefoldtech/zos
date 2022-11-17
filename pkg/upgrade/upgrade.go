package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"

	"github.com/threefoldtech/0-fs/meta"
	"github.com/threefoldtech/0-fs/rofs"
	"github.com/threefoldtech/0-fs/storage"
	"github.com/threefoldtech/zos/pkg/zinit"

	"github.com/rs/zerolog/log"
)

var (
	// ErrRestartNeeded is returned if upgraded requires a restart
	ErrRestartNeeded = fmt.Errorf("restart needed")

	// services that can't be uninstalled with normal procedure
	protected = []string{"identityd", "redis"}

	flistIdentityPath = "/bin/identityd"
)

const (
	defaultHubStorage  = "zdb://hub.grid.tf:9900"
	defaultZinitSocket = "/var/run/zinit.sock"
)

// Upgrader is the component that is responsible
// to keep 0-OS up to date
type Upgrader struct {
	zinit        *zinit.Client
	cache        string
	noSelfUpdate bool
	hub          HubClient
	storage      storage.Storage
}

// UpgraderOption interface
type UpgraderOption func(u *Upgrader) error

// NoSelfUpgrade option
func NoSelfUpgrade(o bool) UpgraderOption {
	return func(u *Upgrader) error {
		u.noSelfUpdate = o

		return nil
	}
}

// Storage option overrides the default hub storage url
func Storage(url string) func(u *Upgrader) error {
	return func(u *Upgrader) error {
		storage, err := storage.NewSimpleStorage(url)
		if err != nil {
			return errors.Wrap(err, "failed to initialize hub storage")
		}
		u.storage = storage
		return nil
	}
}

// Zinit option overrides the default zinit socket
func Zinit(socket string) func(u *Upgrader) error {
	return func(u *Upgrader) error {
		zinit := zinit.New(socket)
		u.zinit = zinit
		return nil
	}
}

// NewUpgrader creates a new upgrader instance
func NewUpgrader(cache string, opts ...UpgraderOption) (*Upgrader, error) {
	u := &Upgrader{
		cache: cache,
	}

	for _, opt := range opts {
		if err := opt(u); err != nil {
			return nil, err
		}
	}

	if u.storage == nil {
		// no storage option was set. use default
		if err := Storage(defaultHubStorage)(u); err != nil {
			return nil, err
		}
	}

	if u.zinit == nil {
		if err := Zinit(defaultZinitSocket)(u); err != nil {
			return nil, err
		}
	}

	return u, nil
}

func (u *Upgrader) flistCache() string {
	return filepath.Join(u.cache, "flist")
}

// Upgrade is the method that does a full upgrade flow
// first check if a new version is available
// if yes, applies the upgrade
// on a successfully update, upgrade WILL NOT RETURN
// instead the upgraded daemon will be completely stopped
func (u *Upgrader) Upgrade(from, to FullFListInfo) error {
	return u.applyUpgrade(from, to)
}

// getFlist accepts fqdn of flist as `<repo>/<name>.flist`
func (u *Upgrader) getFlist(flist string) (meta.Walker, error) {
	db, err := u.hub.Download(u.flistCache(), flist)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download flist")
	}

	store, err := meta.NewStore(db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load flist db")
	}

	walker, ok := store.(meta.Walker)
	if !ok {
		store.Close()
		return nil, errors.Wrap(err, "flist database of unsupported type")
	}

	return walker, nil
}

// InstallBinary from a single flist.
func (u *Upgrader) InstallBinary(flist FListInfo) error {
	log.Info().Str("flist", flist.Fqdn()).Msg("start applying upgrade")

	store, err := u.getFlist(flist.Fqdn())
	if err != nil {
		return errors.Wrapf(err, "failed to process flist: %s", flist.Fqdn())
	}
	defer store.Close()

	if err := u.copyRecursive(store, "/"); err != nil {
		return errors.Wrapf(err, "failed to install flist: %s", flist.Fqdn())
	}

	services, err := u.servicesFromStore(store)
	if err != nil {
		return errors.Wrap(err, "failed to list services from flist")
	}

	return u.ensureRestarted(services...)
}

func (u *Upgrader) servicesFromStore(store meta.Walker) ([]string, error) {
	const zinitPath = "/etc/zinit"

	var services []string
	err := store.Walk(zinitPath, func(path string, info meta.Meta) error {
		if info.IsDir() {
			return nil
		}
		dir := filepath.Dir(path)
		if dir != zinitPath {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}

		services = append(services,
			strings.TrimSuffix(info.Name(), ".yaml"))
		return nil
	})

	if err == meta.ErrNotFound {
		return nil, nil
	}

	return services, err
}

func (u *Upgrader) ensureRestarted(service ...string) error {
	log.Debug().Strs("services", service).Msg("ensure services")
	if len(service) == 0 {
		return nil
	}

	log.Debug().Strs("services", service).Msg("restarting services")
	if err := u.zinit.StopMultiple(20*time.Second, service...); err != nil {
		return err
	}

	for _, name := range service {
		if err := u.zinit.Forget(name); err != nil {
			log.Warn().Err(err).Str("service", name).Msg("could not forget service")
		}

		if err := u.zinit.Monitor(name); err != nil && err != zinit.ErrAlreadyMonitored {
			log.Error().Err(err).Str("service", name).Msg("could not monitor service")
		}
	}

	return nil
}

// UninstallBinary  from a single flist.
func (u *Upgrader) UninstallBinary(flist FListInfo) error {
	return u.uninstall(flist)
}

// upgradeSelf will try to check if the flist has
// an upgraded binary with different revision. If yes
// it will copy the new binary and ask for a restart.
// next time this method is called, it will match the flist
// revision, and hence will continue updating all the other daemons
func (u *Upgrader) upgradeSelf(store meta.Walker) error {
	log.Debug().Msg("starting self upgrade")
	if u.noSelfUpdate {
		log.Debug().Msg("skipping self upgrade")
		return nil
	}

	current := currentRevision()
	log.Debug().Str("revision", current).Msg("current revision")

	bin := currentBinPath()
	info, exists := store.Get(bin)

	if !exists {
		// no bin for update daemon in the flist.
		log.Debug().Str("bin", bin).Msg("binary file does not exist")
		return nil
	}

	newBin := fmt.Sprintf("%s.new", bin)
	if err := u.copyFile(newBin, info); err != nil {
		return err
	}

	// the timeout here is set to 1 min because
	// this most probably will trigger a download
	// of the binary over 0-fs, hence we need to
	// give it enough time to download the file
	// on slow network (i am looking at u Egypt)
	new, err := revisionOf(newBin, 2*time.Minute)
	if err != nil {
		return errors.Wrap(err, "failed to check new update daemon revision number")
	}

	log.Debug().Str("revision", new).Msg("new revision")

	// nothing to be done here.
	if current == new {
		log.Debug().Msg("skipping self upgrade because same revision")
		return nil
	}

	if err := os.Rename(newBin, bin); err != nil {
		return errors.Wrap(err, "failed to update self binary")
	}

	log.Debug().Msg("revisions are differnet, self upgrade is needed")
	return ErrRestartNeeded
}

func (u *Upgrader) uninstall(flist FListInfo) error {
	files, err := flist.Files()
	if err != nil {
		return errors.Wrapf(err, "failed to get list of current installed files for '%s'", flist.Absolute())
	}

	//stop all services names
	var names []string
	for _, file := range files {
		dir := filepath.Dir(file.Path)
		if dir != "/etc/zinit" {
			continue
		}

		name := filepath.Base(file.Path)
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}

		name = strings.TrimSuffix(name, ".yaml")
		// skip self and redis
		if isIn(name, protected) {
			continue
		}

		names = append(names, name)
	}

	log.Debug().Strs("services", names).Msg("stopping services")

	if err = u.zinit.StopMultiple(20*time.Second, names...); err != nil {
		return errors.Wrapf(err, "failed to stop services")
	}

	// we do a forget so any changes of the zinit config
	// themselves get reflected once monitored again
	for _, name := range names {
		if err := u.zinit.Forget(name); err != nil {
			log.Error().Err(err).Str("service", name).Msg("error on zinit forget")
		}
	}

	// now delete ALL files, ignore what doesn't delete
	for _, file := range files {
		stat, err := os.Stat(file.Path)
		if err != nil {
			log.Debug().Err(err).Str("file", file.Path).Msg("failed to check file")
			continue
		}

		if stat.IsDir() {
			continue
		}

		log.Debug().Str("file", file.Path).Msg("deleting file")

		if file.Path == flistIdentityPath {
			log.Debug().Str("file", file.Path).Msg("skip deleting file")
			continue
		}

		if err := os.Remove(file.Path); err != nil {
			log.Error().Err(err).Str("file", file.Path).Msg("failed to remove file")
		}
	}

	return nil
}

func (u *Upgrader) applyUpgrade(from, to FullFListInfo) error {
	log.Info().Str("flist", to.Fqdn()).Str("version", to.TryVersion().String()).Msg("start applying upgrade")

	store, err := u.getFlist(to.Fqdn())
	if err != nil {
		return errors.Wrap(err, "failed to get flist store")
	}

	defer store.Close()

	if err := u.upgradeSelf(store); err != nil {
		return err
	}

	if err := u.uninstall(from.FListInfo); err != nil {
		log.Error().Err(err).Msg("failed to uninstall current flist. Upgraded anyway")
	}

	log.Info().Msg("clean up complete, copying new files")
	services, err := u.servicesFromStore(store)
	if err != nil {
		return err
	}
	if err := u.copyRecursive(store, "/", flistIdentityPath); err != nil {
		return err
	}

	log.Debug().Msg("copying files complete")
	log.Debug().Msg("make sure all services are monitored")
	if err := u.zinit.StartMultiple(20*time.Minute, services...); err != nil {
		return errors.Wrap(err, "failed to monitor services")
	}

	return u.cleanup(services)
}

func (u *Upgrader) cleanup(services []string) error {
	/*
		this is a hack to clean up duplicate services duo to a bug in zinit version 0.2.6 (and earlier)
		the issue has been fixed in next release of zinit (starting 0.2.7)

		the issue is duo to this bug we can end up with multiple running instances of the same service.

		to make sure this is cleaned up as expected we need to do the following:
		- get the pid of each service from zinit status. Those are the processes that are known and
		  managed by zinit
		- for each service, find all processes that is running using the same command
		- for the found service(s), kill the ones that their PIDs does not match the one managed by zinit

		We will always assume that the binary name is the same as the service name
	*/

	names := make(map[string]struct{})
	for _, name := range services {
		names[name] = struct{}{}
	}

	all := make(map[string][]int)
	processes, err := process.Processes()
	if err != nil {
		return err
	}

	for _, ps := range processes {
		cmdline, err := ps.CmdlineSlice()
		if err != nil {
			log.Debug().Err(err).Msgf("failed to parse process '%d' commandline", ps.Pid)
			continue
		}
		if len(cmdline) == 0 {
			continue
		}
		name := cmdline[0]
		if _, ok := names[name]; !ok {
			continue
		}

		all[name] = append(all[name], int(ps.Pid))
	}

	cl := zinit.Default()

	for _, service := range services {
		ps, err := cl.Status(service)
		if errors.Is(err, zinit.ErrUnknownService) {
			continue
		} else if err != nil {
			log.Error().Err(err).Msg("failed to get service status")
		}

		pids, ok := all[service]
		if !ok {
			// probably a short lived service, or has exited
			continue
		}

		for _, pid := range pids {
			if ps.Pid != pid {
				log.Warn().Str("service", service).Int("pid", pid).Msg("found unmanaged process for service. terminating.")
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	}

	return nil
}

func (u *Upgrader) copyRecursive(store meta.Walker, destination string, skip ...string) error {
	return store.Walk("", func(path string, info meta.Meta) error {

		dest := filepath.Join(destination, path)
		if isIn(dest, skip) {
			if info.IsDir() {
				return meta.ErrSkipDir
			}
			log.Debug().Str("file", dest).Msg("skipping file")
			return nil
		}

		if info.IsDir() {
			if err := os.MkdirAll(dest, os.FileMode(info.Info().Access.Mode)); err != nil {
				return err
			}
			return nil
		}

		stat := info.Info()

		switch stat.Type {
		case meta.RegularType:
			// regular file (or other types that we don't handle)
			return u.copyFile(dest, info)
		case meta.LinkType:
			//fmt.Println("link target", stat.LinkTarget)
			target := stat.LinkTarget
			if filepath.IsAbs(target) {
				// if target is absolute, we make sure it's under destination
				// other wise use relative path
				target = filepath.Join(destination, stat.LinkTarget)
			}

			if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
				return err
			}

			return os.Symlink(target, dest)
		default:
			log.Debug().Str("type", info.Info().Type.String()).Msg("ignoring not suppored file type")
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

func (u *Upgrader) copyFile(dst string, src meta.Meta) error {
	log.Info().Str("source", src.Name()).Str("destination", dst).Msg("copy file")

	var (
		isNew  = false
		dstOld string
	)

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		// case where this is a new file
		// we just need to copy from flist to root
		isNew = true
	}

	var err error
	if !isNew {
		dstOld = dst + ".old"
		if err := os.Rename(dst, dstOld); err != nil {
			return err
		}

		defer func() {
			if err == nil {
				if err := os.Remove(dstOld); err != nil {
					log.Error().Err(err).Str("file", dstOld).Msg("failed to clean up backup file")
				}
				return
			}

			if err := os.Rename(dstOld, dst); err != nil {
				log.Error().Err(err).Str("file", dst).Msg("failed to restore file after a failed download")
			}
		}()
	}

	downloader := rofs.NewDownloader(u.storage, src)
	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, os.FileMode(src.Info().Access.Mode))
	if err != nil {
		return err
	}
	defer fDst.Close()

	if err = downloader.Download(fDst); err != nil {
		return err
	}

	return nil
}
