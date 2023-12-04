package perf

import (
	"context"
	"time"
)

type Task interface {
	ID() string
	Cron() string
	Description() string
	Jitter() time.Duration
	Run(ctx context.Context) (interface{}, error)
}
