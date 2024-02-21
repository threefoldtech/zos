package app

import (
	"path/filepath"

	"github.com/pkg/errors"
)

// defaultBootedPath is the path where to store the booted flag
const defaultBootedPath = "/var/run/modules"

// MarkBooted creates a file in a memory
// this file then can be used to check if "something" has been restared
// if its the first time it starts
func MarkBooted(name string) error {
	return markBooted(name, defaultBootedPath, defaultFS)
}

func markBooted(name string, bootedPath string, fs fileSystem) error {
	if err := fs.MkdirAll(bootedPath, 0770); err != nil {
		return err
	}
	path := filepath.Join(bootedPath, name)
	marker, err := fs.Create(path)
	if err != nil {
		return errors.Wrapf(err, "failed to mark service as booted: %s", name)
	}

	return marker.Close()
}

// IsFirstBoot checks if the a file has been created by MarkBooted function
func IsFirstBoot(name string) bool {
	return isFirstBoot(name, defaultBootedPath, defaultFS)
}

func isFirstBoot(name string, bootedPath string, fs fileSystem) bool {
	path := filepath.Join(bootedPath, name)
	_, err := fs.Stat(path)
	return err != nil
}
