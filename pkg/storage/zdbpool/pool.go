package zdbpool

import (
	"io/ioutil"
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

// New creates a ZDBPool with path
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
func (p *ZDBPool) Reserved() (uint64, error) {
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

// Namespace gets a namespace info from pool
func (p *ZDBPool) Namespace(name string) (info NSInfo, err error) {
	path := filepath.Join(p.path, name, "zdb-namespace")
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()
	var header Header
	if err := ReadHeader(f, &header); err != nil {
		return info, errors.Wrapf(err, "failed to read namespace header at %s", path)
	}

	info = NSInfo{
		Name: name,
		Size: header.MaxSize,
	}

	return
}

// Namespaces returns a list of NSinfo of all the namespace present in the pool
func (p *ZDBPool) Namespaces() ([]NSInfo, error) {

	dirs, err := ioutil.ReadDir(p.path)
	if err != nil {
		return nil, err
	}

	var namespaces []NSInfo
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		if dir.Name() == "default" {
			continue
		}

		info, err := p.Namespace(dir.Name())
		if os.IsNotExist(err) {
			// not a valid namespace directory
			continue
		} else if err != nil {
			return nil, err
		}

		namespaces = append(namespaces, info)
	}

	return namespaces, nil
}

// Exists checks if a namespace exists in the pool or not
// this method is way faster then using Namespaces cause it doesn't have to read any data
func (p *ZDBPool) Exists(name string) bool {
	path := filepath.Join(p.path, name, "zdb-namespace")

	_, err := os.Stat(path)
	return err == nil
}
