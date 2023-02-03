package namespace

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

var (
	ErrNotNamedNamespace = fmt.Errorf("name space is not named")
)

const (
	netNSPath = "/var/run/netns"
)

// Create creates a new persistent (bind-mounted) network namespace and returns an object
// representing that namespace, without switching to it.
func Create(name string) (ns.NetNS, error) {

	// Create the directory for mounting network namespaces
	// This needs to be a shared mountpoint in case it is mounted in to
	// other namespaces (containers)
	err := os.MkdirAll(netNSPath, 0755)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create network namespace directory %s", netNSPath)
	}

	// Remount the namespace directory shared. This will fail if it is not
	// already a mountpoint, so bind-mount it on to itself to "upgrade" it
	// to a mountpoint.
	err = unix.Mount("", netNSPath, "none", unix.MS_SHARED|unix.MS_REC, "")
	if err != nil {
		if err != unix.EINVAL {
			return nil, fmt.Errorf("mount --make-rshared %s failed: %q", netNSPath, err)
		}

		// Recursively remount /var/run/netns on itself. The recursive flag is
		// so that any existing netns bindmounts are carried over.
		err = unix.Mount(netNSPath, netNSPath, "none", unix.MS_BIND|unix.MS_REC, "")
		if err != nil {
			return nil, fmt.Errorf("mount --rbind %s %s failed: %q", netNSPath, netNSPath, err)
		}

		// Now we can make it shared
		err = unix.Mount("", netNSPath, "none", unix.MS_SHARED|unix.MS_REC, "")
		if err != nil {
			return nil, fmt.Errorf("mount --make-rshared %s failed: %q", netNSPath, err)
		}

	}

	// create an empty file at the mount point
	nsPath := path.Join(netNSPath, name)
	mountPointFd, err := os.Create(nsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create file %s", nsPath)
	}
	mountPointFd.Close()

	// Ensure the mount point is cleaned up on errors; if the namespace
	// was successfully mounted this will have no effect because the file
	// is in-use
	defer os.RemoveAll(nsPath)

	var wg sync.WaitGroup
	wg.Add(1)

	// do namespace work in a dedicated goroutine, so that we can safely
	// Lock/Unlock OSThread without upsetting the lock/unlock state of
	// the caller of this function
	go (func() {
		defer wg.Done()
		runtime.LockOSThread()
		// Don't unlock. By not unlocking, golang will kill the OS thread when the
		// goroutine is done (for go1.10+)

		var origNS ns.NetNS
		origNS, err = ns.GetNS(getCurrentThreadNetNSPath())
		if err != nil {
			return
		}
		defer origNS.Close()

		// create a new netns on the current thread
		err = unix.Unshare(unix.CLONE_NEWNET)
		if err != nil {
			return
		}

		// Put this thread back to the orig ns, since it might get reused (pre go1.10)
		defer func() {
			_ = origNS.Set()
		}()

		// bind mount the netns from the current thread (from /proc) onto the
		// mount point. This causes the namespace to persist, even when there
		// are no threads in the ns.
		err = unix.Mount(getCurrentThreadNetNSPath(), nsPath, "none", unix.MS_BIND, "")
		if err != nil {
			err = fmt.Errorf("failed to bind mount ns at %s: %v", nsPath, err)
		}
	})()
	wg.Wait()

	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %v", err)
	}

	return ns.GetNS(nsPath)
}

func getCurrentThreadNetNSPath() string {
	// /proc/self/ns/net returns the namespace of the main thread, not
	// of whatever thread this goroutine is running on.  Make sure we
	// use the thread's net namespace since the thread is switching around
	return fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), unix.Gettid())
}

// Delete deletes a network namespace
func Delete(ns ns.NetNS) error {
	if ns == nil {
		return fmt.Errorf("invalid namespace")
	}
	if err := ns.Close(); err != nil {
		return err
	}
	nsPath := ns.Path()
	// Only unmount if it's been bind-mounted (don't touch namespaces in /proc...)
	if strings.HasPrefix(nsPath, netNSPath) {
		if err := unix.Unmount(nsPath, unix.MNT_DETACH|unix.MNT_FORCE); err != nil {
			return fmt.Errorf("failed to unmount NS: at %s: %v", nsPath, err)
		}

		if err := os.Remove(nsPath); err != nil {
			return fmt.Errorf("failed to remove ns path %s: %v", nsPath, err)
		}
	}

	return nil
}

// Name gets the name of the namespace if it was created with Create
// otherwise return ErrNotNamedNamespace
func Name(ns ns.NetNS) (string, error) {
	path := ns.Path()

	dir, name := filepath.Split(path)
	if dir != netNSPath {
		return "", ErrNotNamedNamespace
	}

	return name, nil
}

// Exists checks if a network namespace exists or not
func Exists(name string) bool {
	nsPath := filepath.Join(netNSPath, name)
	_, err := os.Stat(nsPath)
	if err != nil {
		return false
	}
	mounted, err := isNamespaceMounted(name)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if namespace is mounted")
		return false
	}

	if !mounted {
		//the file shouldn't be there
		_ = os.Remove(nsPath)
	}

	return mounted
}

// GetByName return a namespace by its name
func GetByName(name string) (ns.NetNS, error) {
	nsPath := filepath.Join(netNSPath, name)
	n, err := ns.GetNS(nsPath)
	var ne ns.NSPathNotExistErr
	if errors.As(err, &ne) {
		return nil, os.ErrNotExist
	}

	return n, err
}

// List returns a list of all the names of the network namespaces
func List(prefix string) ([]string, error) {
	infos, err := os.ReadDir(netNSPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read network namespace directory: %w", err)
	}

	names := make([]string, 0, len(infos))
	for _, info := range infos {
		if info.IsDir() {
			continue
		}

		if prefix != "" && !strings.HasPrefix(info.Name(), prefix) {
			continue
		}

		names = append(names, info.Name())
	}

	return names, nil
}

func isNamespaceMounted(name string) (bool, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return false, errors.Wrap(err, "failed to list mounts")
	}
	defer file.Close()

	path := filepath.Join(netNSPath, name)
	mounts := bufio.NewScanner(file)
	for mounts.Scan() {
		line := mounts.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		// we are looking for line that looks like
		// nsfs /var/run/netns/<name> nsfs rw 0 0
		if parts[0] != "nsfs" {
			// we searching for nsfs type only
			continue
		}

		if parts[1] == path {
			return true, nil
		}
	}

	return false, nil
}
