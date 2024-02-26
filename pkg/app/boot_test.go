package app

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

// TestMarkBooted tests the markBooted function against multiple scenarios.
// it tests different scenarios of failures during directory creation or file creation.
func TestMarkBooted(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("markBooted failed to MkdirAll", func(t *testing.T) {
		exec := &pkg.TestExecuter{}
		errMkdir := fmt.Errorf("failed to MkdirAll")
		exec.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(fmt.Errorf("failed to MkdirAll"))

		err := markBooted(testFile, bootedPath, exec)
		require.Error(t, err)
		require.Equal(t, err, errMkdir)
		exec.AssertExpectations(t)
	})

	t.Run("markBooted failed to Create", func(t *testing.T) {
		exec := &pkg.TestExecuter{}
		exec.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&pkg.FSTestExecuter{}, fmt.Errorf("couldn't create file with name file1"))

		err := markBooted(testFile, bootedPath, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("markBooted valid file", func(t *testing.T) {
		exec := &pkg.TestExecuter{}
		exec.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&pkg.FSTestExecuter{}, nil)

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
		exec := &pkg.TestExecuter{}
		exec.On("Stat", testFilePath).
			Return(nil, nil)

		firstBoot := isFirstBoot(testFile, bootedPath, exec)
		require.False(t, firstBoot)
		exec.AssertExpectations(t)
	})

	t.Run("the file is not first booted", func(t *testing.T) {
		exec := &pkg.TestExecuter{}
		exec.On("Stat", testFilePath).
			Return(nil, fmt.Errorf("couldn't find file in the bootedPath"))

		firstBoot := isFirstBoot(testFile, bootedPath, exec)
		require.True(t, firstBoot)
		exec.AssertExpectations(t)
	})
}
