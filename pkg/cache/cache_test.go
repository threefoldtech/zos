package cache

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testExecuter used to mock the filesystem object
type testExecuter struct {
	mock.Mock
}

func (t *testExecuter) MkdirAll(path string, perm fs.FileMode) error {
	args := t.Called(path, perm)
	return args.Error(0)
}

func (t *testExecuter) Mkdir(path string, perm fs.FileMode) error {
	args := t.Called(path, perm)
	return args.Error(0)
}

func (t *testExecuter) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	args := t.Called(source, target, fstype, flags, data)
	return args.Error(0)
}

// TestVolatileDir tests the volatileDir function against multiple scenarios.
// it tests both scenarios of the volatileDir failed/succeeded to be mounted
func TestVolatileDir(t *testing.T) {
	name := "test"
	var size uint64 = 1024
	const volatileBaseDir = "/var/run/cache"

	t.Run("volatileDir failed to Mount", func(t *testing.T) {
		exec := &testExecuter{}

		filePath := filepath.Join(volatileBaseDir, name)

		exec.On("MkdirAll", volatileBaseDir, fs.FileMode(0700)).
			Return(nil)

		exec.On("Mkdir", filePath, fs.FileMode(0700)).
			Return(nil)

		exec.On("Mount", "none", filePath, "tmpfs", uintptr(0), fmt.Sprintf("size=%d", size)).
			Return(fmt.Errorf("failed to Mount"))

		_, err := volatileDir(name, size, exec, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})
	t.Run("volatileDir Mounted successfully", func(t *testing.T) {
		exec := &testExecuter{}

		filePath := filepath.Join(volatileBaseDir, name)

		exec.On("MkdirAll", volatileBaseDir, fs.FileMode(0700)).
			Return(nil)

		exec.On("Mkdir", filePath, fs.FileMode(0700)).
			Return(nil)

		exec.On("Mount", "none", filePath, "tmpfs", uintptr(0), fmt.Sprintf("size=%d", size)).
			Return(nil)

		n, err := volatileDir(name, size, exec, exec)
		require.NoError(t, err)
		require.Equal(t, n, filePath)
		exec.AssertExpectations(t)
	})
}
