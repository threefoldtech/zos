package modules

//go:generate rm -rf stubs
//go:generate mkdir stubs

//go:generate zbus -module flist -version 0.0.1 -name flist -package stubs github.com/threefoldtech/zosv2/modules+Flister stubs/flist_stub.go
type Flister interface {
	// Mount mounts an flist located at url using the 0-db located at storage.
	// Returns the path in the filesystem where the flist is mounted or an error
	Mount(url string, storage string) (path string, err error)

	// Umount the flist mounted at path
	Umount(path string) error
}
