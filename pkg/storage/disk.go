package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

const (
	vdiskVolumeName = "vdisks"
)

type vdiskModule struct {
	path string
}

// NewVDiskModule creates a new disk allocator
func NewVDiskModule(v pkg.VolumeAllocater) (pkg.VDiskModule, error) {
	path, err := v.Path(vdiskVolumeName)
	if errors.Is(err, os.ErrNotExist) {
		path, err = v.CreateFilesystem(vdiskVolumeName, 0, pkg.SSDDevice)
	}

	if err != nil {
		return nil, err
	}

	return &vdiskModule{path: filepath.Clean(path)}, nil
}

// AllocateDisk with given size, return path to virtual disk (size in MB)
func (d *vdiskModule) Allocate(id string, size int64) (string, error) {
	path := filepath.Join(d.path, id)

	if _, err := os.Stat(path); err == nil {
		// file exists
		return path, errors.Wrapf(os.ErrExist, "disk with id '%s' already exists", id)
	}

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	return path, syscall.Fallocate(int(file.Fd()), 0, 0, size*1024*1024)
}

// DeallocateVDisk removes a virtual disk
func (d *vdiskModule) Deallocate(id string) error {
	path := filepath.Join(d.path, id)
	// this to avoid passing an `injection` id like '../name'
	// and end up deleting a file on the system. so only delete
	// allocated disks
	location := filepath.Dir(path)
	if filepath.Clean(location) != d.path {
		return fmt.Errorf("invalid disk id: '%s'", id)
	}

	return os.Remove(path)
}
