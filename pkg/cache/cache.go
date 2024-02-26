package cache

import (
	"fmt"
	"path/filepath"

	"github.com/threefoldtech/zos/pkg"
)

const (
	Megabyte = 1024 * 1024
)

// VolatileDir creates a new cache directory that is stored on a tmpfs.
// This means data stored in this directory will NOT survive a reboot.
// Use this when you need to store data that needs to survive deamon reboot but not between reboots
// It is the caller's responsibility to remove the directory when no longer needed.
// If the directory already exist error of type os.IsExist will be returned
func VolatileDir(name string, size uint64) (string, error) {
	return volatileDir(name, size, pkg.DefaultSystemOS, pkg.DefaultSysCall)
}

func volatileDir(name string, size uint64, os pkg.SystemOS, syscall pkg.SystemCall) (string, error) {
	const volatileBaseDir = "/var/run/cache"
	name = filepath.Join(volatileBaseDir, name)
	if err := os.MkdirAll(volatileBaseDir, 0700); err != nil {
		return name, err
	}

	if err := os.Mkdir(name, 0700); err != nil {
		return name, err
	}
	return name, syscall.Mount("none", name, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}
