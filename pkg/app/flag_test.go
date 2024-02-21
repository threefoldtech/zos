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

	var testFS TestExecuter
	testFS.On("MkdirAll", flagsDir, os.ModePerm).
		Return(nil)

	testFS.On("Create", testFilePath).
		Return(&testReadWriteCloser{}, nil)

	err := setFlag(testFile, flagsDir, &testFS)
	require.NoError(t, err)
	testFS.AssertExpectations(t)
}

func TestCheckFlag(t *testing.T) {
	testFile := "test"
	flagsDir := "tmp/flags"
	testFilePath := filepath.Join(flagsDir, testFile)

	var testFS TestExecuter

	testFS.On("Stat", testFilePath).
		Return(nil, fs.ErrNotExist)

	testFS.On("IsNotExist", fs.ErrNotExist).
		Return(true)

	flagExists := checkFlag(testFile, flagsDir, &testFS)
	require.False(t, flagExists)
	testFS.AssertExpectations(t)
}

func TestDeleteFlag(t *testing.T) {
	key := LimitedCache
	err := DeleteFlag(key)
	require.NoError(t, err)
}

func TestDeleteBadFlag(t *testing.T) {
	key := "/"
	err := DeleteFlag(key)
	require.Error(t, err)
}
