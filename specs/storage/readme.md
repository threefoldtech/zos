# Storage module

This module responsible to manage everything related with storage. In 0-OS we have 2 different storage primitives, storage pool and [0-db](https://github.com/threefoldtech/0-db).

Storage pool are used when a direct disk access is required. Typical example would be a container needs to persist some data on disk.
A storage pool is an abstract concept, it just represent something that has a path and on which we can write files.
Usually this is implement using some kind of filesystem on top of one or multiple disks.

[0-DB](https://github.com/threefoldtech/0-db) is the other storage primitives, it provide an efficient Key-Value store interface on top of a disk. 0-db implements different running mode that can server different use cases.

The storage module itself is spitted into 3 sub module, each one responsible for a specific tasks.

List of sub-modules:

- [disks](#disk-sub-module)
- [storage pools](#storage-pool-sub-module)
- [0-db](#0-db-sub-module)


## Disk sub-module

Responsible to discover and prepare all the disk available on a node to be ready to use for the other sub-modules

### Interface

```go
type DiskModule interface {
    // walk over all the disks of the node
    // create a partition table
    // create a single partition that takes the full size of the disk
    Prepare() (DiskInfo[],error)

    // Will wipe completely all the content of a disk
    // this is done by overwriting the partition table with random data
    Wipe(deviceName string) (error)

    // Starts a process that will monitor if some disks are added/removed during the
    // lifetime of the node. This can be used by a higher layer to propagate event up the stack
    // if some application needs to be made aware of disks being added/removed
    Monitor(ctx context.Context) (<-DiskEvent)

    // TODO:  to be be further defined, but we will need something that can give us the exact amount of storage
    // a node has. This is used to compute the resource unit provided by the node.
    Stats() (diskStats, error)
}
```

## Storage pool sub-module

### Interface
Responsible to do the capacity planning of the storage pools on top of the disk prepare by the disk sub-module

```go
type StoragePool interface{
    ID() string
    Path() string
    Size() string
}

type StoragePoolModule interface {
    // Reserve creates a storage space that can be used to mount
    // as a volume into a container/VM
    // Reserve is responsible to choose the best suited storage pool to use
    Reserve(diskType DiskType, size int64) (StoragePool, error)
    // Release releases a storage pool acquired by Reserve
    // after this the storage pool need to be considered null and should not be used anymore.
    Release(storagePool) (error)
}
```

## 0-db sub-module
Responsible to do the capacity planning of the 0-db on top of the disk prepare by the disk sub-module

### Interface

```go
type ZDBNamespace struct {
    Name string
    DiskType DiskType
    Size int64
    mode ZdbMode
    Password string
    Port int // Listen port of the 0-db owning the namespace
}

type ZDB struct {
    DiskType DiskType
    Size int64
    mode ZdbMode
    AdminPassword string
    Port int // Listen port of the 0-db process
}

type ZDBModule interface {
    // ReserveNamespace reserve a 0-db namespace of specified size, type and mode
    // ReserveNamespace is responsible for the capacity planning and must decide what is the
    // best place where to create this namespace. If can happen that ReserveNamespace endup
    // deploying a new 0-db to be able to create the namespace
    ReserveNamespace(diskType DiskType, size int64, mode ZdbMode, password string) (ZDBNamespace, error)

    // ReleaseNamespace delete a 0-db namespace acquired by ReserveNamespace
    ReleaseNamespace(ZDBNamespace) (error)

    // ReserveZdb will always create a new 0-db process
    // ReserveZdb is responsible for the capacity planning and must decide what is the
    // best disk where to deploy this 0-db.
    ReserveZdb(diskType DiskType, size int64, mode ZdbMode, adminPassword string) (ZDB, error)
    // ReleaseZdb stop the 0-db process and delete all its data.
    ReleaseNamespace(zdb ZDB) (error)
}
```
