package app

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	flagsDir     = "/tmp/flags"
	LimitedCache = "limited-cache"
)

// MakeFlag is used when the /var/cache cannot be mounted on a SSD or HDD,
// it will mount the cache disk on a temporary file system in memory.
func SetFlag(key string) error {
	// creating a file that will be used as a flag
	// the flag can 'warn' other deamons that the /var/cache is not on HDD or SDD
	if err := os.MkdirAll(flagsDir, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to create flags directory")
	}

	f, err := os.Create(filepath.Join(flagsDir, key))
	if err != nil {
		return errors.Wrap(err, "failed to create the flag file")
	}

	return f.Close()
}

func ClearFlag(key string) error {
	if err := os.Remove(filepath.Join(flagsDir, key)); err != nil {
		return errors.Wrap(err, "failed to remove flag file")
	}
	return nil
}

func CheckFlag(key string) bool {
	_, err := os.Stat(filepath.Join(flagsDir, key))
	return os.IsNotExist(err)
}
