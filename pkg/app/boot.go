package app

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// bootedPath is the path where to store the booted flag
const bootedPath = "/var/run/modules"

// bootManager contains the path of the booted files and fileSystem
type bootManager struct {
	bootedPath string
	fs         fileSystem
}

type fileSystem interface {
	Create(string) (io.ReadCloser, error)
	MkdirAll(string, uint32) error
	Stat(string) (any, error)
}

type defaultBootManager struct{}

func (fs defaultBootManager) Create(path string) (io.ReadCloser, error) {
	return os.Create(path)
}

func (dfs defaultBootManager) MkdirAll(path string, perm uint32) error {
	return os.MkdirAll(path, fs.FileMode(perm))
}

func (dfs defaultBootManager) Stat(path string) (any, error) {
	return os.Stat(path)
}

func defaultMarkManager() *bootManager {
	return &bootManager{
		fs:         defaultBootManager{},
		bootedPath: bootedPath,
	}
}

// MarkBooted creates a file in a memory
// this file then can be used to check if "something" has been restared
// if its the first time it starts
func MarkBooted(name string) error {
	manager := defaultMarkManager()
	return manager.markBooted(name)
}

func (m *bootManager) markBooted(name string) error {
	if err := m.fs.MkdirAll(bootedPath, 0770); err != nil {
		return err
	}
	path := filepath.Join(bootedPath, name)
	marker, err := m.fs.Create(path)
	if err != nil {
		return errors.Wrapf(err, "failed to mark service as booted: %s", name)
	}

	return marker.Close()
}

// IsFirstBoot checks if the a file has been created by MarkBooted function
func IsFirstBoot(name string) bool {
	manager := defaultMarkManager()
	return manager.isFirstBoot(name)
}

func (m *bootManager) isFirstBoot(name string) bool {
	path := filepath.Join(bootedPath, name)
	_, err := m.fs.Stat(path)
	return err != nil
}
