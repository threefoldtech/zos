package logger

import (
	"encoding/json"
	"io/fs"
	"os"
)

// RedisType defines redis logger type name
const RedisType = "redis"

// FileType defines file logger type name
const FileType = "file"

// ConsoleType defines console logger type name
const ConsoleType = "console"

// Logs defines a custom backend with variable settings
type Logs struct {
	Type string    `json:"type"`
	Data LogsRedis `json:"data"`
}

// LogsRedis defines how to connect a redis logs backend
type LogsRedis struct {
	// Stdout is the redis url for stdout (redis://host/channel)
	Stdout string `json:"stdout"`

	// Stderr is the redis url for stderr (redis://host/channel)
	Stderr string `json:"stderr"`
}

type marshaler interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

var (
	defMarshaler    = &defaultMarshaler{}
	defReaderWriter = &defaultReaderWriter{}
)

type defaultMarshaler struct{}

func (m *defaultMarshaler) Marshal(val any) ([]byte, error) {
	return json.Marshal(val)
}

func (m *defaultMarshaler) Unmarshal(data []byte, str any) error {
	return json.Unmarshal(data, str)
}

type readerWriter interface {
	WriteFile(string, []byte, fs.FileMode) error
	ReadFile(string) ([]byte, error)
}

type defaultReaderWriter struct{}

func (rw *defaultReaderWriter) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (rw *defaultReaderWriter) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Serialize dumps logs array into a json file
func Serialize(path string, logs []Logs) error {
	return serialize(path, logs, defMarshaler, defReaderWriter)
}

func serialize(path string, logs []Logs, m marshaler, rw readerWriter) error {
	data, err := m.Marshal(logs)
	if err != nil {
		return err
	}

	return rw.WriteFile(path, data, 0644)
}

// Deserialize reads json from disks and returns []Logs
func Deserialize(path string) ([]Logs, error) {
	return deserialize(path, defMarshaler, defReaderWriter)
}

func deserialize(path string, m marshaler, rw readerWriter) ([]Logs, error) {
	logs := []Logs{}

	data, err := rw.ReadFile(path)
	if err != nil {
		return logs, err
	}

	err = m.Unmarshal(data, &logs)
	return logs, err
}
