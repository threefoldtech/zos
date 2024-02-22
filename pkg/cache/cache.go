package cache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

const (
	Megabyte = 1024 * 1024
)

var defaultFS = &defaultFileSystem{}

type fileSystem interface {
	MkdirAll(string, fs.FileMode) error
	Mkdir(string, fs.FileMode) error
}

type defaultFileSystem struct{}

func (dfs *defaultFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (dfs *defaultFileSystem) Mkdir(path string, perm fs.FileMode) error {
	return os.Mkdir(path, perm)
}

var defaultSys = &defaultSysCall{}

type systemCall interface {
	Mount(string, string, string, uintptr, string) error
}

type defaultSysCall struct{}

func (dfs *defaultSysCall) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return syscall.Mount(source, target, fstype, flags, data)
}

// VolatileDir creates a new cache directory that is stored on a tmpfs.
// This means data stored in this directory will NOT survive a reboot.
// Use this when you need to store data that needs to survive deamon reboot but not between reboots
// It is the caller's responsibility to remove the directory when no longer needed.
// If the directory already exist error of type os.IsExist will be returned
func VolatileDir(name string, size uint64) (string, error) {
	return volatileDir(name, size, defaultFS, defaultSys)
}

func volatileDir(name string, size uint64, fs fileSystem, syscall systemCall) (string, error) {
	const volatileBaseDir = "/var/run/cache"
	name = filepath.Join(volatileBaseDir, name)
	if err := fs.MkdirAll(volatileBaseDir, 0700); err != nil {
		return name, err
	}

	if err := fs.Mkdir(name, 0700); err != nil {
		return name, err
	}
	return name, syscall.Mount("none", name, "tmpfs", 0, fmt.Sprintf("size=%d", size))
}
