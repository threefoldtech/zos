package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetFlag(t *testing.T) {
	testFile := "test"
	flagsDir := "tmp/flags"
	testFilePath := filepath.Join(flagsDir, testFile)

	t.Run("setFlag invalid flag", func(t *testing.T) {
		var exec TestExecuter
		exec.On("MkdirAll", flagsDir, os.ModePerm).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&testReadWriteCloser{}, nil)

		err := setFlag(testFile, flagsDir, &exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("setFlag valid flag", func(t *testing.T) {
		var exec TestExecuter
		exec.On("MkdirAll", flagsDir, os.ModePerm).
			Return(nil)

		exec.On("Create", testFilePath).
			Return(&testReadWriteCloser{}, nil)

		err := setFlag(testFile, flagsDir, &exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})
}

func TestCheckFlag(t *testing.T) {
	testFile := "test"
	flagsDir := "tmp/flags"
	testFilePath := filepath.Join(flagsDir, testFile)

	t.Run("checkFlag flag does not exist", func(t *testing.T) {
		var exec TestExecuter

		exec.On("Stat", testFilePath).
			Return(nil, nil)

		exec.On("IsNotExist", nil).
			Return(false)

		flagExists := checkFlag(testFile, flagsDir, &exec)
		require.True(t, flagExists)
		exec.AssertExpectations(t)
	})

	t.Run("checkFlag flag exist", func(t *testing.T) {
		var exec TestExecuter

		exec.On("Stat", testFilePath).
			Return(nil, fs.ErrNotExist)

		exec.On("IsNotExist", fs.ErrNotExist).
			Return(true)

		flagExists := checkFlag(testFile, flagsDir, &exec)
		require.False(t, flagExists)
		exec.AssertExpectations(t)
	})
}

func TestDeleteFlag(t *testing.T) {
	t.Run("delete flag invalid cache type", func(t *testing.T) {
		key := "/"
		flagsDir := "tmp/flags"

		var exec TestExecuter
		err := deleteFlag(key, flagsDir, &exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("delete flag valid cache type", func(t *testing.T) {
		key := LimitedCache
		flagsDir := "tmp/flags"

		var exec TestExecuter
		exec.On("RemoveAll", filepath.Join(flagsDir, key)).
			Return(nil)

		err := deleteFlag(key, flagsDir, &exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})
}
