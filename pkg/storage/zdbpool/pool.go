package zdbpool

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Prefix is a string used as prefix in the filesystem volume used
// to storge 0-db namespaces
const Prefix = "zdb"

// ZDBPool represent a part of a disk that is reserved to store 0-db data
type ZDBPool struct {
	path string
}

func New(path string) ZDBPool {
	return ZDBPool{path}
}

// NSInfo is a struct containing information about a 0-db namespace
type NSInfo struct {
	Name string
	Size uint32
}

// Reserved return the amount of storage that has been reserved by all the
// namespace in the pool
func (p ZDBPool) Reserved() (uint64, error) {
	ns, err := p.Namespaces()
	if err != nil {
		return 0, err
	}

	var total uint64
	for _, n := range ns {
		total += uint64(n.Size)
	}

	return total, nil
}

// Namespaces returns a list of NSinfo of all the namespace present in the pool
func (p ZDBPool) Namespaces() ([]NSInfo, error) {
	root := p.path
	nsinfos := make([]NSInfo, 0)
	header := Header{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if path == root {
			return nil
		}

		// skip default namespace
		if info.Name() == "default" {
			return filepath.SkipDir
		}

		hp := filepath.Join(path, "zdb-namespace")
		f, err := os.Open(hp)
		if err != nil {
			return errors.Wrapf(err, "failed to open namespace header at %s", path)
		}
		defer f.Close()

		if err := ReadHeader(f, &header); err != nil {
			return errors.Wrapf(err, "failed to read namespace header at %s", path)
		}

		nsinfos = append(nsinfos, NSInfo{
			Name: info.Name(),
			Size: header.MaxSize,
		})

		return filepath.SkipDir
	})

	return nsinfos, err
}

// Exists checks if a namespace exists in the pool or not
// this method is way faster then using Namespaces cause it doesn't have to read any data
func (p ZDBPool) Exists(name string) bool {
	path := filepath.Join(p.path, name)

	_, err := os.Stat(path)
	return err == nil
}
