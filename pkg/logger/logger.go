package logger

// ContainerLogger defines a logging backend
type ContainerLogger interface {
	// Send stdout line to the backend
	Stdout(line string) error

	// Send stderr line to the backend
	Stderr(line string) error

	// Close stdout handler
	CloseStdout()

	// Close stderr handler
	CloseStderr()
}

// Logs defines a custom backend with variable settings
type Logs struct {
	Type string    `json:"type"`
	Data LogsRedis `json:"data"`
}

// LogsRedis defines how to connect a redis logs backend
type LogsRedis struct {
	Endpoint string `json:"endpoint"`
	Channel  string `json:"channel"`
}
