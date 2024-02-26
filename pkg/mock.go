package pkg

import (
	"io"
	"io/fs"

	"github.com/stretchr/testify/mock"
)

// TestExecuter used to mock the filesystem object
type TestExecuter struct {
	mock.Mock
}

func (exec *TestExecuter) Create(path string) (io.ReadCloser, error) {
	args := exec.Called(path)
	return args.Get(0).(*FSTestExecuter), args.Error(1)
}

func (exec *TestExecuter) MkdirAll(path string, perm fs.FileMode) error {
	args := exec.Called(path, perm)
	return args.Error(0)
}

func (exec *TestExecuter) Mkdir(path string, perm fs.FileMode) error {
	args := exec.Called(path, perm)
	return args.Error(0)
}

func (exec *TestExecuter) Stat(path string) (any, error) {
	args := exec.Called(path)
	return args.Get(0), args.Error(1)
}

func (exec *TestExecuter) IsNotExist(err error) bool {
	args := exec.Called(err)
	return args.Bool(0)
}

func (exec *TestExecuter) RemoveAll(path string) error {
	args := exec.Called(path)
	return args.Error(0)
}

func (t *TestExecuter) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	args := t.Called(source, target, fstype, flags, data)
	return args.Error(0)
}

type FSTestExecuter struct {
	io.Reader
}

func (t *FSTestExecuter) Close() error {
	return nil
}
