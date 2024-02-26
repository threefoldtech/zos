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
		os := &pkg.SystemOSMock{}
		errMkdir := fmt.Errorf("failed to MkdirAll")
		os.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(fmt.Errorf("failed to MkdirAll"))

		err := markBooted(testFile, bootedPath, os)
		require.Error(t, err)
		require.Equal(t, err, errMkdir)
		os.AssertExpectations(t)
	})

	t.Run("markBooted failed to Create", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		os.On("Create", testFilePath).
			Return(&pkg.FSMock{}, fmt.Errorf("couldn't create file with name file1"))

		err := markBooted(testFile, bootedPath, os)
		require.Error(t, err)
		os.AssertExpectations(t)
	})

	t.Run("markBooted valid file", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("MkdirAll", bootedPath, fs.FileMode(0770)).
			Return(nil)

		os.On("Create", testFilePath).
			Return(&pkg.FSMock{}, nil)

		err := markBooted(testFile, bootedPath, os)
		require.NoError(t, err)
		os.AssertExpectations(t)
	})
}

// TestIsFirstBoo tests the isFirstBoot function against multiple scenarios.
// it tests both scenarios of the file being first booted or not
func TestIsFirstBoot(t *testing.T) {
	testFile := "test"
	bootedPath := "var/run/modules"
	testFilePath := filepath.Join(bootedPath, testFile)

	t.Run("the file is first booted", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("Stat", testFilePath).
			Return(nil, nil)

		firstBoot := isFirstBoot(testFile, bootedPath, os)
		require.False(t, firstBoot)
		os.AssertExpectations(t)
	})

	t.Run("the file is not first booted", func(t *testing.T) {
		os := &pkg.SystemOSMock{}
		os.On("Stat", testFilePath).
			Return(nil, fmt.Errorf("couldn't find file in the bootedPath"))

		firstBoot := isFirstBoot(testFile, bootedPath, os)
		require.True(t, firstBoot)
		os.AssertExpectations(t)
	})
}
