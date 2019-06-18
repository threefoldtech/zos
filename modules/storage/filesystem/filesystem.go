package filesystem

import (
	"context"

	"github.com/threefoldtech/zosv2/modules"
)

// Volume represents a logical volume in the pool. Volumes can be nested
type Volume interface {
	Path() string
	Volumes() ([]Volume, error)
	AddVolume(name string) (Volume, error)
	RemoveVolume(name string) error
	Size() (uint64, error)
	Limit(size uint64) error
}

// Pool represents a created filesystem
type Pool interface {
	Volume
	Name() string
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
