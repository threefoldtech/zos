package app

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestExecuter used to mock the filesystem object
type testExecuter struct {
	mock.Mock
}

func (t *testExecuter) Create(path string) (io.ReadCloser, error) {
	args := t.Called(path)
	return args.Get(0).(*testReadWriteCloser), args.Error(1)
}

func (t *testExecuter) MkdirAll(path string, perm fs.FileMode) error {
	args := t.Called(path, perm)
	return args.Error(0)
}

func (t *testExecuter) Stat(path string) (any, error) {
	args := t.Called(path)
	return args.Get(0), args.Error(1)
}

func (t *testExecuter) IsNotExist(err error) bool {
	args := t.Called(err)
	return args.Bool(0)
}

func (t *testExecuter) RemoveAll(path string) error {
	args := t.Called(path)
	return args.Error(0)
}

type testReadWriteCloser struct {
	io.Reader
}

func (t *testReadWriteCloser) Close() error {
	return nil
}

// TestMarkBooted tests the markBooted function against multiple scenarios.
// it tests different scenarios of failures during directory creation or file creation.
func TestMarkBooted(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("markBooted failed to MkdirAll", func(t *testing.T) {
		var exec testExecuter
		errMkdir := fmt.Errorf("failed to MkdirAll")
		exec.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(fmt.Errorf("failed to MkdirAll"))

		err := markBooted(testFile, bootedPath, &exec)
		require.Error(t, err)
		require.Equal(t, err, errMkdir)
		exec.AssertExpectations(t)
	})

	t.Run("markBooted failed to Create", func(t *testing.T) {
		exec := &testExecuter{}
		exec.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&testReadWriteCloser{}, fmt.Errorf("couldn't create file with name file1"))

		err := markBooted(testFile, bootedPath, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("markBooted valid file", func(t *testing.T) {
		exec := &testExecuter{}
		exec.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&testReadWriteCloser{}, nil)

		err := markBooted(testFile, bootedPath, exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})
}

// TestIsFirstBoo tests the isFirstBoot function against multiple scenarios.
// it tests both scenarios of the file being first booted or not
func TestIsFirstBoot(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("the file is first booted", func(t *testing.T) {
		exec := &testExecuter{}
		exec.On("Stat", testFilePath).
			Return(nil, nil)

		firstBoot := isFirstBoot(testFile, bootedPath, exec)
		require.False(t, firstBoot)
		exec.AssertExpectations(t)
	})

	t.Run("the file is not first booted", func(t *testing.T) {
		exec := &testExecuter{}
		exec.On("Stat", testFilePath).
			Return(nil, fmt.Errorf("couldn't find file in the bootedPath"))

		firstBoot := isFirstBoot(testFile, bootedPath, exec)
		require.True(t, firstBoot)
		exec.AssertExpectations(t)
	})
}
