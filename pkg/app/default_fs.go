package app

import (
	"io"
	"io/fs"
	"os"
)

var defaultFS = &defaultFileSystem{}

type fileSystem interface {
	Create(string) (io.ReadCloser, error)
	MkdirAll(string, fs.FileMode) error
	Stat(string) (any, error)
	IsNotExist(error) bool
	RemoveAll(string) error
}

type defaultFileSystem struct{}

func (dfs *defaultFileSystem) Create(path string) (io.ReadCloser, error) {
	return os.Create(path)
}

func (dfs *defaultFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (dfs *defaultFileSystem) Stat(path string) (any, error) {
	return os.Stat(path)
}

func (dfs *defaultFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (dfs *defaultFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}
