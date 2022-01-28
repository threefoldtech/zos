package flist

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/environment"
)

const (
	defaultRoot = "/var/cache/modules/flist"
)

const mib = 1024 * 1024

type commander interface {
	Command(name string, arg ...string) *exec.Cmd
}

type cmd func(name string, arg ...string) *exec.Cmd

func (c cmd) Command(name string, args ...string) *exec.Cmd {
	return c(name, args...)
}

type flistModule struct {
	// root directory where all
	// the working file of the module will be located
	root string

	// underneath are the path for each
	// sub folder used by the flist module
	flist      string
	cache      string
	mountpoint string
	pid        string
	log        string
	run        string

	storage   pkg.VolumeAllocater
	commander commander
}

func newFlister(root string, storage pkg.VolumeAllocater, commander commander) pkg.Flister {
	if root == "" {
		root = defaultRoot
	}

	if err := os.MkdirAll(root, 0755); err != nil {
		panic(err)
	}

	// ensure we have proper permission for existing directory
	if err := os.Chmod(root, 0755); err != nil {
		panic(err)
	}

	// prepare directory layout for the module
	for _, path := range []string{"flist", "cache", "mountpoint", "pid", "log", "run"} {
		p := filepath.Join(root, path)
		if err := os.MkdirAll(p, 0755); err != nil {
			panic(err)
		}
		// ensure we have proper permission for existing directory
		if err := os.Chmod(p, 0755); err != nil {
			panic(err)
		}
	}

	return &flistModule{
		root:       root,
		flist:      filepath.Join(root, "flist"),
		cache:      filepath.Join(root, "cache"),
		mountpoint: filepath.Join(root, "mountpoint"),
		pid:        filepath.Join(root, "pid"),
		log:        filepath.Join(root, "log"),
		run:        filepath.Join(root, "run"),

		storage:   storage,
		commander: commander,
	}
}

type options []string

func (o options) Find(k string) int {
	for i, x := range o {
		if strings.EqualFold(x, k) {
			return i
		}
	}

	return -1
}

// New creates a new flistModule
func New(root string, storage pkg.VolumeAllocater) pkg.Flister {
	return newFlister(root, storage, cmd(exec.Command))
}

// NamedMount implements the Flister.NamedMount interface
func (f *flistModule) NamedMount(name, url, storage string, opts pkg.MountOptions) (string, error) {
	return f.mount(name, url, storage, opts)
}

// Mount implements the Flister.Mount interface
func (f *flistModule) Mount(url, storage string, opts pkg.MountOptions) (string, error) {
	rnd, err := random()
	if err != nil {
		return "", errors.Wrap(err, "failed to generate random id for the mount")
	}
	return f.mount(rnd, url, storage, opts)
}

func (f *flistModule) mountpath(name string) (string, error) {
	mountpath := filepath.Join(f.mountpoint, name)
	if filepath.Dir(mountpath) != f.mountpoint {
		return "", errors.New("invalid mount name")
	}

	return mountpath, nil
}

// ErrAlreadyMounted is returned when checking if a path has already
// something mounted on it
var ErrAlreadyMounted = errors.New("path is already mounted")

// valid checks that this mount path is free, and can be used
func (f *flistModule) valid(path string) error {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to check mountpoint: %s", path)
	}

	if !stat.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	if err := exec.Command("mountpoint", path).Run(); err == nil {
		return ErrAlreadyMounted
	}

	return nil
}

func (f *flistModule) mount(name, url, storage string, opts pkg.MountOptions) (string, error) {
	sublog := log.With().Str("url", url).Str("storage", storage).Logger()
	sublog.Info().Msg("request to mount flist")

	mountpoint, err := f.mountpath(name)
	if err != nil {
		return "", err
	}

	err = f.valid(mountpoint)
	if errors.Is(err, ErrAlreadyMounted) {
		// if everything is already in place, just early return
		log.Info().Msgf("flist is already mounted at %s, nothing more to do", mountpoint)
		return mountpoint, nil
	} else if err != nil {
		return "", err
	}

	env, err := environment.Get()
	if err != nil {
		return "", errors.Wrap(err, "failed to parse node environment")
	}

	if storage == "" {
		storage = env.FlistURL
	}

	flistPath, err := f.downloadFlist(url)
	if err != nil {
		sublog.Err(err).Msg("fail to download flist")
		return "", err
	}

	var backend pkg.Filesystem
	var newAllocation bool
	var args []string
	if !opts.ReadOnly {
		sublog.Info().Msgf("check if subvolume %s already exists", name)
		// check if the filesystem doesn't already exists
		backend, err = f.storage.Path(name)
		if err != nil {
			sublog.Info().Msgf("create new subvolume %s", name)
			// and only create a new one if it doesn't exist
			if opts.Limit == 0 || len(opts.Type) == 0 {
				// sanity check in case type is not set always use hdd
				return "", fmt.Errorf("invalid mount option, missing disk type and/or size")
			}
			newAllocation = true
			backend, err = f.storage.CreateFilesystem(name, opts.Limit*mib, opts.Type)
			if err != nil {
				return "", errors.Wrap(err, "failed to create read-write subvolume for 0-fs")
			}
		}
	}

	if len(backend.Path) != 0 {
		args = append(args, "-backend", backend.Path)
		// in case of an error (mount is never fully completed)
		// we need to deallocate the filesystem
		defer func() {
			if newAllocation && err != nil {
				_ = f.storage.ReleaseFilesystem(name)
			}
		}()
	} else {
		args = append(args, "-ro")
	}

	if err != nil {
		return "", errors.Wrap(err, "invalid mount point")
	}

	if err = os.MkdirAll(mountpoint, 0755); err != nil {
		return "", err
	}
	pidPath := filepath.Join(f.pid, name) + ".pid"
	logPath := filepath.Join(f.log, name) + ".log"

	args = append(args,
		"-cache", f.cache,
		"-meta", flistPath,
		"-storage-url", storage,
		"-daemon",
		"-pid", pidPath,
		"-log", logPath,
		mountpoint,
	)
	sublog.Info().Strs("args", args).Msg("starting 0-fs daemon")
	cmd := f.commander.Command("g8ufs", args...)

	var out []byte
	if out, err = cmd.CombinedOutput(); err != nil {
		sublog.Err(err).Str("out", string(out)).Msg("fail to start 0-fs daemon")
		return "", err
	}

	// wait for the daemon to be ready
	// we check the pid file is created
	pid, err := waitPidFile(time.Second*5, pidPath, true)
	if err != nil {
		sublog.Error().Err(err).Msg("pid file of 0-fs daemon not created")
		return "", err
	}

	// set oom adj
	if err := setOOMAdj(pid, -17); err != nil {
		log.Error().Err(err).Int64("pid", pid).Msg("failed to set 0-fs oom adjust for pid")
		// this won't affect the operation, it can make
		// the 0-fs process vulnerable to oom, so we continue anyway
	}

	// and scan the logs after "mount ready"
	if err := waitMountedLog(time.Second*5, logPath); err != nil {
		sublog.Error().Err(err).Msg("0-fs daemon did not start properly")
		return "", err
	}
	// the track file is a symlink to the process pid
	// if the link is broken, then the fs has exited gracefully
	// otherwise we can get the fs pid from the track path
	// if the pid does not exist of the target does not exist
	// we can clean up the named mount
	trackPath := filepath.Join(f.run, name)
	if err = os.Symlink(pidPath, trackPath); err != nil {
		sublog.Error().Err(err).Msg("failed track fs pid")
	}

	return mountpoint, nil
}

func (f *flistModule) getPid(pidPath string) (int64, error) {
	pid, err := ioutil.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseInt(string(pid), 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "invalid pid value in file, expected int")
	}

	return value, nil
}

func (f *flistModule) getMountOptions(pidPath string) (options, error) {
	pid, err := f.getPid(pidPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get pid from file: %s", pidPath)
	}

	return f.getMountOptionsForPID(pid)
}

func (f *flistModule) getMountOptionsForPID(pid int64) (options, error) {
	cmdline, err := ioutil.ReadFile(path.Join("/proc", fmt.Sprint(pid), "cmdline"))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read mount (%d) cmdline", pid)
	}

	parts := bytes.Split(cmdline, []byte{0})

	result := make(options, 0, len(parts))
	for _, part := range parts {
		result = append(result, string(part))
	}

	return result, nil
}

func (f *flistModule) HashFromRootPath(name string) (string, error) {
	base := filepath.Base(name)
	pidPath := filepath.Join(f.pid, base) + ".pid"

	opts, err := f.getMountOptions(pidPath)
	if err != nil {
		return "", err
	}

	for _, opt := range opts {
		// if option start with the flist meta path
		if strings.HasPrefix(opt, f.flist) {
			// extracting hash (dirname) from argument
			return filepath.Base(opt), nil
		}
	}

	return "", fmt.Errorf("could not find rootfs hash name")
}

// NamedUmount implements the Flister.NamedUmount interface
func (f *flistModule) NamedUmount(name string) error {
	return f.Umount(filepath.Join(f.mountpoint, name))
}

// Umount implements the Flister.Umount interface
func (f *flistModule) Umount(path string) error {
	log.Info().Str("path", path).Msg("request unmount flist")

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("specified path is not a directory")
	}

	if filepath.Dir(path) != filepath.Clean(f.mountpoint) {
		return fmt.Errorf("trying to unmount a directory outside of the flist module boundaries")
	}

	_, name := filepath.Split(path)
	pidPath := filepath.Join(f.pid, name) + ".pid"

	// from here on out, skip errors because we want to try to clean up as much as possible
	opts, err := f.getMountOptions(pidPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to read mount options")

		//we don't return an error in case the file is gone somehow
		//but the mount is still there. So better to fail on next call
		//than leaking
	}

	if err := syscall.Unmount(path, syscall.MNT_DETACH); err != nil {
		log.Error().Err(err).Str("path", path).Msg("fail to umount flist")
	}

	if _, err := waitPidFile(time.Second*2, pidPath, false); err != nil {
		log.Error().Err(err).Str("path", path).Msg("0-fs daemon did not stop properly")

		pid, err := f.getPid(pidPath)
		if err == nil {
			if err := forceStop(int(pid)); err != nil {
				log.Error().Int64("pid", pid).Err(err).Msg("failed to kill 0-fs process")
			}
		} else {
			log.Error().Int64("pid", pid).Err(err).Msg("failed to get pid")
		}

	}

	// clean up working dirs
	for _, path := range []string{pidPath, path} {
		if err := os.RemoveAll(path); err != nil {
			log.Error().Err(err).Msgf("fail to remove '%s'", path)
		}
	}

	// clean up subvolume should be done only for RW mounts.
	if opts.Find("-ro") >= 0 {
		return nil
	}

	if err := f.storage.ReleaseFilesystem(name); err != nil {
		log.Error().Err(err).Msg("fail to clean up subvolume")
	}

	return nil
}

// FlistHash returns md5 of flist if available (requesting the hub)
func (f *flistModule) FlistHash(url string) (string, error) {
	// first check if the md5 of the flist is available
	md5URL := url + ".md5"
	resp, err := http.Get(md5URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		hash, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		cleanhash := strings.TrimSpace(string(hash))
		return cleanhash, nil
	}

	return "", fmt.Errorf("fail to fetch hash, response: %v", resp.StatusCode)
}

// downloadFlist downloads an flits from a URL
// if the flist location also provide and md5 hash of the flist
// this function will use it to avoid downloading an flist that is
// already present locally
func (f *flistModule) downloadFlist(url string) (string, error) {
	// first check if the md5 of the flist is available
	hash, err := f.FlistHash(url)
	if err == nil {
		flistPath := filepath.Join(f.flist, strings.TrimSpace(string(hash)))
		f, err := os.Open(flistPath)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		defer f.Close()

		if err == nil {
			log.Info().Str("url", url).Msg("flist already in on the filesystem")
			// flist is already present locally, verify it's still valid
			equal, err := md5Compare(hash, f)
			if err != nil {
				return "", err
			}
			if equal {
				return flistPath, nil
			}
			log.Info().Str("url", url).Msg("flist on filesystem is corrupted, re-downloading it")
			// if not equal the rest of the function will overwrite the faulty flist
		} else {
			log.Info().Str("url", url).Msg("flist not in cache, downloading")
		}
	}

	// we don't have the flist locally yet, let's download it
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("fail to download flist: %v", resp.Status)
	}

	return f.saveFlist(resp.Body)
}

// saveFlist save the flist contained in r
// it save the flist by its md5 hash
// to avoid loading the full flist in memory to compute the hash
// it uses a MultiWriter to write the flist in a temporary file and fill up
// the md5 hash then it rename the file to the hash
func (f *flistModule) saveFlist(r io.Reader) (string, error) {
	tmp, err := ioutil.TempFile(f.flist, "*_flist_temp")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	h := md5.New()
	mr := io.MultiWriter(tmp, h)
	if _, err := io.Copy(mr, r); err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", h.Sum(nil))
	path := filepath.Join(f.flist, hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}

	return path, nil
}

func md5Compare(hash string, r io.Reader) (bool, error) {
	h := md5.New()
	_, err := io.Copy(h, r)
	if err != nil {
		return false, err
	}
	return strings.Compare(fmt.Sprintf("%x", h.Sum(nil)), hash) == 0, nil
}

func random() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return fmt.Sprintf("%x", b), err
}

func setOOMAdj(pid int64, adj int) error {
	if err := ioutil.WriteFile(filepath.Join("/proc/", fmt.Sprint(pid), "oom_adj"), []byte(fmt.Sprint(adj)), 0644); err != nil {
		return errors.Wrapf(err, "failed to update oom priority for (PID: %d)", pid)
	}

	return nil
}

// waitPidFile wait for a file pointed by path to be created or deleted
// for at most timeout duration
// is exists is true, it waits for the file to exists
// else it waits for the file to be deleted
func waitPidFile(timeout time.Duration, path string, exists bool) (int64, error) {
	const delay = time.Millisecond * 100

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

loop:
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			_, err := os.Stat(path)
			// check exist but the file is not there yet,
			// or check NOT exist but it is still there
			// we try again
			if (exists && os.IsNotExist(err)) || (!exists && err == nil) {
				time.Sleep(delay)
			} else if err != nil && !os.IsNotExist(err) {
				//another error that is NOT IsNotExist.
				return 0, err
			} else {
				break loop
			}
		}
	}

	if !exists {
		return 0, nil
	}

	// otherwise, we need to try reading the pid of this 0-fs process
	// from the pid file.
	// note that the file might exists but the pid is not written yet.
	// hence we need to retry reading this.
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			pidBytes, err := ioutil.ReadFile(path)
			if err != nil {
				//file should exist, so this error is perminant
				return 0, err
			}

			if len(pidBytes) == 0 {
				time.Sleep(delay)
				continue
			}
			pid, err := strconv.ParseInt(strings.TrimSpace(string(pidBytes)), 10, 64)
			if err != nil {
				// possible that the pid is not fully written. we can try again
				time.Sleep(delay)
				continue
			}

			return pid, nil
		}
	}
}

func waitMountedLog(timeout time.Duration, logfile string) error {
	const target = "mount ready"
	const delay = time.Millisecond * 500

	f, err := os.Open(logfile)
	if err != nil {
		return err
	}
	defer f.Close()
	br := bufio.NewReader(f)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// this goroutine looks for "mount ready"
	// in the logs of the 0-fs
	cErr := make(chan error)
	go func(ctx context.Context, r io.Reader, cErr chan<- error) {
		for {
			select {
			case <-ctx.Done():
				// ensure we don't leak the goroutine
				cErr <- ctx.Err()
			default:
				line, err := br.ReadString('\n')
				if err != nil {
					time.Sleep(delay)
					continue
				}

				if !strings.Contains(line, target) {
					time.Sleep(delay)
					continue
				}
				// found
				cErr <- nil
				return
			}
		}
	}(ctx, br, cErr)

	return <-cErr
}

func forceStop(pid int) error {
	slog := log.With().Int("pid", pid).Logger()
	slog.Info().Msg("trying to force stop by killing the process")

	p, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}

	if err := p.Signal(syscall.SIGTERM); err != nil {
		slog.Error().Err(err).Msg("failed to send SIGTERM to process")
	}

	time.Sleep(time.Second)
	if pidExist(p) {
		slog.Info().Msgf("process didn't stop gracefully, lets kill it")
		if err := p.Signal(syscall.SIGKILL); err != nil {
			slog.Error().Err(err).Msg("failed to send SIGKILL to process")
		}
	}

	return nil
}

func pidExist(p *os.Process) bool {
	// https://github.com/golang/go/issues/14146#issuecomment-176888204
	return p.Signal(syscall.Signal(0)) == nil
}

var _ pkg.Flister = (*flistModule)(nil)
