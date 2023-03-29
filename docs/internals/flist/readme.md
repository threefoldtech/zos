# Flist module

## Zbus

Flist module is available on zbus over the following channel:

| module | object | version |
|--------|--------|---------|
|flist   |[flist](#public-interface)| 0.0.1

## Home Directory
flist keeps some data in the following locations:
| directory | path|
|----|---|
| root| `/var/cache/modules/containerd`|

## Introduction

This module is responsible to "mount an flist" in the filesystem of the node. The mounted directory contains all the files required by containers or (in the future) VMs.

The flist module interface is very simple. It does not expose any way to choose where to mount the flist or have any reference to containers or VM. The only functionality is to mount a given flist and receive the location where it is mounted. It is up to the above layer to do something useful with this information.

The flist module itself doesn't contain the logic to understand the flist format or to run the fuse filesystem. It is just a wrapper that manages [0-fs](https://github.com/threefoldtech/0-fs) processes.

Its only job is to download the flist, prepare the isolation of all the data and then start 0-fs with the proper arguments.

## Public interface [![GoDoc](https://godoc.org/github.com/threefoldtech/zos/pkg/flist?status.svg)](https://godoc.org/github.com/threefoldtech/zos/pkg/flist)

```go

//Flister is the interface for the flist module
type Flister interface {
	// Mount mounts an flist located at url using the 0-db located at storage
	// in a RO mode. note that there is no way u can unmount a ro flist because
	// it can be shared by many users, it's then up to system to decide if the
	// mount is not needed anymore and clean it up
	Mount(name, url string, opt MountOptions) (path string, err error)

	// UpdateMountSize change the mount size
	UpdateMountSize(name string, limit gridtypes.Unit) (path string, err error)

	// Umount a RW mount. this only unmounts the RW layer and remove the assigned
	// volume.
	Unmount(name string) error

	// HashFromRootPath returns flist hash from a running g8ufs mounted with NamedMount
	HashFromRootPath(name string) (string, error)

	// FlistHash returns md5 of flist if available (requesting the hub)
	FlistHash(url string) (string, error)

	Exists(name string) (bool, error)
}

```

## zinit unit

The zinit unit file of the module specifies the command line, test command, and the order in which the services need to be booted.

Flist module depends on the storage and network pkg.
This is because it needs connectivity to download flist and data and it needs storage to be able to cache the data once downloaded.

Flist doesn't do anything special on the system except creating a bunch of directories it will use during its lifetime.
