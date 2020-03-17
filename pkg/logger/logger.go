package logger

// Define a logging backend
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

type Logs struct {
	Type string    `json:"type"`
	Data LogsRedis `json:"data"`
}

type LogsRedis struct {
	Endpoint string `json:"endpoint"`
	Channel  string `json:"channel"`
}
