package pkg

import (
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

//go:generate mkdir -p stubs

//go:generate zbusc -module flist -version 0.0.1 -name flist -package stubs github.com/threefoldtech/zos/pkg+Flister stubs/flist_stub.go

var (
	//DefaultMountOptions has sane values for mount
	DefaultMountOptions = MountOptions{
		ReadOnly: false,
		Limit:    256 * gridtypes.Megabyte, //Mib
		Type:     zos.SSDDevice,
	}

	//ReadOnlyMountOptions shortcut for readonly mount options
	ReadOnlyMountOptions = MountOptions{
		ReadOnly: true,
	}
)

// MountOptions struct
type MountOptions struct {
	// ReadOnly
	ReadOnly bool
	// Limit size of read-write layer
	Limit gridtypes.Unit
	// Type of disk to use
	Type zos.DeviceType
}

//Flister is the interface for the flist module
type Flister interface {
	// Mount mounts an flist located at url using the 0-db located at storage
	// in a RO mode. note that there is no way u can unmount a ro flist because
	// it can be shared by many users, it's then up to system to decide if the
	// mount is not needed anymore and clean it up
	MountRO(url string, storage string) (path string, err error)

	// Mounts an flist in rw mode. the name given should be granted by the user to
	// be unique. if a similar name exists, the current path will be returned.
	MountRW(name, url, storage string, size gridtypes.Unit) (path string, err error)

	// Umount a RW mount. this only unmounts the RW layer and remove the assigned
	// volume.
	Unmount(name string) error

	// HashFromRootPath returns flist hash from a running g8ufs mounted with NamedMount
	HashFromRootPath(name string) (string, error)

	// FlistHash returns md5 of flist if available (requesting the hub)
	FlistHash(url string) (string, error)
}
