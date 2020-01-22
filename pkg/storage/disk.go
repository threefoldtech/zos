package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
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
func (d *vdiskModule) Allocate(size int64) (string, error) {
	name := uuid.New().String()
	path := filepath.Join(d.path, name)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	return path, syscall.Fallocate(int(file.Fd()), 0, 0, size*1024*1024)
}

// DeallocateVDisk removes a virtual disk
func (d *vdiskModule) Deallocate(path string) error {
	location := filepath.Dir(path)
	if filepath.Clean(location) != d.path {
		return fmt.Errorf("invalid disk path")
	}

	return os.Remove(path)
}
