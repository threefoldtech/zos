package healthcheck

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/perf"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	id          = "healthcheck"
	schedule    = "0 */20 * * * *"
	description = "health check task runs multiple checks to ensure the node is in a usable state and set flags for the power daemon to stop reporting uptime if it is not usable"
)

// NewTask returns a new health check task.
func NewTask() perf.Task {
	checks := []checkFunc{
		cacheCheck,
	}
	return &healthcheckTask{
		checks: checks,
		errors: make(map[string][]string),
	}
}

type checkFunc func(context.Context) (string, error)

type healthcheckTask struct {
	checks []checkFunc
	errors map[string][]string
}

var _ perf.Task = (*healthcheckTask)(nil)

// ID returns task ID.
func (h *healthcheckTask) ID() string {
	return id
}

// Cron returns task cron schedule.
func (h *healthcheckTask) Cron() string {
	return schedule
}

// Description returns task description.
func (h *healthcheckTask) Description() string {
	return description
}

// Run executes the health checks.
func (h *healthcheckTask) Run(ctx context.Context) (interface{}, error) {
	log.Debug().Msg("starting health check task")
	for k := range h.errors {
		// reset errors on each run
		h.errors[k] = make([]string, 0)
	}

	for _, check := range h.checks {
		label, err := check(ctx)
		if err == nil {
			continue
		}
		h.errors[label] = append(h.errors[label], err.Error())
	}

	cl := perf.GetZbusClient(ctx)
	zui := stubs.NewZUIStub(cl)

	for label, data := range h.errors {
		zui.PushErrors(ctx, label, data)
	}
	return h.errors, nil
}

func cacheCheck(ctx context.Context) (string, error) {
	const label = "cache"
	const checkFile = "/var/cache/healthcheck"

	_, err := os.Create(checkFile)
	if err != nil {
		if err := app.SetFlag(app.ReadonlyCache); err != nil {
			log.Error().Err(err).Msg("failed to set readonly flag")
		}
		return label, fmt.Errorf("failed to write to cache: %w", err)
	}
	defer os.Remove(checkFile)

	if err := app.DeleteFlag(app.ReadonlyCache); err != nil {
		log.Error().Err(err).Msg("failed to delete readonly flag")
	}
	return label, nil
}
