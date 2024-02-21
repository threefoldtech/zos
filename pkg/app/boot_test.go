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

type TestExecuter struct {
	mock.Mock
}

func (t *TestExecuter) Create(path string) (io.ReadCloser, error) {
	args := t.Called(path)
	return args.Get(0).(*testReadWriteCloser), args.Error(1)
}

func (t *TestExecuter) MkdirAll(path string, perm fs.FileMode) error {
	args := t.Called(path, perm)
	return args.Error(0)
}

func (t *TestExecuter) Stat(path string) (any, error) {
	args := t.Called(path)
	return args.Get(0), args.Error(1)
}

func (t *TestExecuter) IsNotExist(err error) bool {
	args := t.Called(err)
	return args.Bool(0)
}

func (t *TestExecuter) RemoveAll(path string) error {
	args := t.Called(path)
	return args.Error(0)
}

type testReadWriteCloser struct {
	io.Reader
}

func (t *testReadWriteCloser) Close() error {
	return nil
}

func TestMarkBooted(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("markBooted invalid file", func(t *testing.T) {
		var testFS TestExecuter
		testFS.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		testFS.On("Create", testFilePath).
			Return(&testReadWriteCloser{}, fmt.Errorf("couldn't create file with name file1"))

		err := markBooted(testFile, bootedPath, &testFS)
		require.Error(t, err)
		testFS.AssertExpectations(t)
	})

	t.Run("markBooted a valid file", func(t *testing.T) {
		var testFS TestExecuter
		testFS.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		testFS.On("Create", testFilePath).
			Return(&testReadWriteCloser{}, nil)

		err := markBooted(testFile, bootedPath, &testFS)
		require.NoError(t, err)
		testFS.AssertExpectations(t)
	})
}

func TestIsFirstBoot(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("the file is first booted", func(t *testing.T) {
		var testFS TestExecuter
		testFS.On("Stat", testFilePath).
			Return(nil, nil)

		firstBoot := isFirstBoot(testFile, bootedPath, &testFS)
		require.False(t, firstBoot)
		testFS.AssertExpectations(t)
	})

	t.Run("the file is not first booted", func(t *testing.T) {
		var testFS TestExecuter
		testFS.On("Stat", testFilePath).
			Return(nil, fmt.Errorf("couldn't find file in the bootedPath"))

		firstBoot := isFirstBoot(testFile, bootedPath, &testFS)
		require.True(t, firstBoot)
		testFS.AssertExpectations(t)
	})
}
