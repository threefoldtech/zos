package pkg

import (
	"io"
	"io/fs"
	"os"
	"syscall"
)

// instance of default filesystem
var DefaultSystemOS = &defaultFileSystem{}

// SystemOS interface to mock actual OS operations.
type SystemOS interface {
	Create(string) (io.ReadCloser, error)
	MkdirAll(string, fs.FileMode) error
	Mkdir(string, fs.FileMode) error
	Stat(string) (any, error)
	RemoveAll(string) error
	IsNotExist(error) bool
}

type defaultFileSystem struct{}

func (dfs *defaultFileSystem) Create(path string) (io.ReadCloser, error) {
	return os.Create(path)
}

func (dfs *defaultFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (dfs *defaultFileSystem) Mkdir(path string, perm fs.FileMode) error {
	return os.Mkdir(path, perm)
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

var DefaultSysCall = &defaultSysCall{}

type SystemCall interface {
	Mount(string, string, string, uintptr, string) error
}

type defaultSysCall struct{}

func (dfs *defaultSysCall) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return syscall.Mount(source, target, fstype, flags, data)
}
