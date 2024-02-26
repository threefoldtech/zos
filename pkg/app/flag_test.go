package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg"
)

// TestSetFlag tests the setFlag function against multiple scenarios.
// it tests both scenarios of failure/success in setting the flag
func TestSetFlag(t *testing.T) {
	testFile := "test"
	flagsDir := "tmp/flags"
	testFilePath := filepath.Join(flagsDir, testFile)

	t.Run("setFlag invalid flag", func(t *testing.T) {
		exec := &pkg.TestExecuter{}
		exec.On("MkdirAll", flagsDir, os.ModePerm).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&pkg.FSTestExecuter{}, fmt.Errorf("failed to create file"))

		err := setFlag(testFile, flagsDir, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("setFlag valid flag", func(t *testing.T) {
		exec := &pkg.TestExecuter{}
		exec.On("MkdirAll", flagsDir, os.ModePerm).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&pkg.FSTestExecuter{}, nil)

		err := setFlag(testFile, flagsDir, exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})
}

// TestCheckFlag tests the checkFlag function against multiple scenarios.
// it tests both scenarios of flag exist/not exist
func TestCheckFlag(t *testing.T) {
	testFile := "test"
	flagsDir := "tmp/flags"
	testFilePath := filepath.Join(flagsDir, testFile)

	t.Run("checkFlag flag exist", func(t *testing.T) {
		exec := &pkg.TestExecuter{}

		exec.On("Stat", testFilePath).
			Return(nil, nil)

		exec.On("IsNotExist", nil).
			Return(false)

		flagExists := checkFlag(testFile, flagsDir, exec)
		require.True(t, flagExists)
		exec.AssertExpectations(t)
	})

	t.Run("checkFlag flag does not exist", func(t *testing.T) {
		exec := &pkg.TestExecuter{}

		exec.On("Stat", testFilePath).
			Return(nil, fs.ErrNotExist)

		exec.On("IsNotExist", fs.ErrNotExist).
			Return(true)

		flagExists := checkFlag(testFile, flagsDir, exec)
		require.False(t, flagExists)
		exec.AssertExpectations(t)
	})
}

// TestDeleteFlag tests the deleteFlag function against multiple scenarios.
// it tests both scenarios of valid/invalid cache type
func TestDeleteFlag(t *testing.T) {
	t.Run("delete invalid cache type", func(t *testing.T) {
		key := "/"
		flagsDir := "tmp/flags"

		exec := &pkg.TestExecuter{}
		err := deleteFlag(key, flagsDir, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("delete valid cache type", func(t *testing.T) {
		key := LimitedCache
		flagsDir := "tmp/flags"

		exec := &pkg.TestExecuter{}
		exec.On("RemoveAll", filepath.Join(flagsDir, key)).
			Return(nil)

		err := deleteFlag(key, flagsDir, exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})
}
