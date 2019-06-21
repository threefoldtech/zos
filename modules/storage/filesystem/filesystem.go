package filesystem

import (
	"context"

	"github.com/threefoldtech/zosv2/modules"
)

// Usage struct
type Usage struct {
	Size uint64
	Used uint64
}

// Volume represents a logical volume in the pool. Volumes can be nested
type Volume interface {
	Path() string
	Volumes() ([]Volume, error)
	AddVolume(name string) (Volume, error)
	RemoveVolume(name string) error
	Usage() (Usage, error)
	Limit(size uint64) error
	Name() string
	FsType() string
}

// Pool represents a created filesystem
type Pool interface {
	Volume
	Mounted() (string, bool)
	Mount() (string, error)
	UnMount() error
	AddDevice(device string) error
	RemoveDevice(device string) error

	// Health() ?
}

// Filesystem defines a filesystem interface
type Filesystem interface {
	Create(ctx context.Context, name string, devices []string, profile modules.RaidProfile) (Pool, error)
	List(ctx context.Context) ([]Pool, error)
}
