package filesystem

import (
	"context"

	"github.com/threefoldtech/zosv2/modules"
)

// Volume represents a logical volume in the pool. Volumes can be nested
type Volume interface {
	Volumes() ([]Volume, error)
	AddVolume(size uint64) (Volume, error)
	RemoveVolume(volume Volume) error
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
	Create(ctx context.Context, name string, devices []string, policy modules.RaidProfile) (Pool, error)
}
