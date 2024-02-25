package logger

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testExecuter struct {
	mock.Mock
}

func (exec *testExecuter) Marshal(val any) ([]byte, error) {
	args := exec.Called(val)
	return args.Get(0).([]byte), args.Error(1)
}

func (exec *testExecuter) Unmarshal(data []byte, str any) error {
	args := exec.Called(data, str)
	return args.Error(0)
}

func (exec *testExecuter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	args := exec.Called(path, data, perm)
	return args.Error(0)
}

func (exec *testExecuter) ReadFile(path string) ([]byte, error) {
	args := exec.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func TestSerialize(t *testing.T) {
	path := "var/tmp/path"
	logs := []Logs{}
	data := []byte{}

	t.Run("serialize failed to marshal", func(t *testing.T) {
		exec := &testExecuter{}

		exec.On("Marshal", logs).
			Return([]byte{}, fmt.Errorf("failed to marshal"))

		err := serialize(path, logs, exec, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("serialize failed to write file", func(t *testing.T) {
		exec := &testExecuter{}

		exec.On("Marshal", logs).
			Return([]byte{}, nil)

		exec.On("WriteFile", path, data, fs.FileMode(0644)).
			Return(fmt.Errorf("failed to write file"))

		err := serialize(path, logs, exec, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("serialize logs to path", func(t *testing.T) {
		exec := &testExecuter{}

		exec.On("Marshal", logs).
			Return([]byte{}, nil)

		exec.On("WriteFile", path, data, fs.FileMode(0644)).
			Return(nil)

		err := serialize(path, logs, exec, exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})
}

func TestDeserialize(t *testing.T) {
	path := "var/tmp/path"
	logs := []Logs{}
	data := []byte{}

	t.Run("deserialize failed to read file", func(t *testing.T) {
		exec := &testExecuter{}

		exec.On("ReadFile", path).
			Return([]byte{}, fmt.Errorf("failed to read file"))

		_, err := deserialize(path, exec, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("deserialize failed to Unmarshal", func(t *testing.T) {
		exec := &testExecuter{}

		exec.On("ReadFile", path).
			Return([]byte{}, nil)

		exec.On("Unmarshal", data, &logs).
			Return(fmt.Errorf("failed to Unmarshal"))

		_, err := deserialize(path, exec, exec)
		require.Error(t, err)
		exec.AssertExpectations(t)
	})

	t.Run("deserialize path", func(t *testing.T) {
		exec := &testExecuter{}

		exec.On("ReadFile", path).
			Return([]byte{}, nil)

		exec.On("Unmarshal", data, &logs).
			Return(nil)

		_, err := deserialize(path, exec, exec)
		require.NoError(t, err)
		exec.AssertExpectations(t)
	})
}
