package logger

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
