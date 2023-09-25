package perf

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/utils"
)

// PerformanceMonitor holds the module data
type PerformanceMonitor struct {
	scheduler *gocron.Scheduler
	pool      *redis.Pool
	tasks     []Task
}

// NewPerformanceMonitor returns PerformanceMonitor instance
func NewPerformanceMonitor(redisAddr string) (*PerformanceMonitor, error) {
	redisPool, err := utils.NewRedisPool(redisAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating new redis pool")
	}

	scheduler := gocron.NewScheduler(time.UTC)

	return &PerformanceMonitor{
		scheduler: scheduler,
		pool:      redisPool,
		tasks:     []Task{},
	}, nil
}

// AddTask a simple helper method to add new tasks
func (pm *PerformanceMonitor) AddTask(task Task) {
	pm.tasks = append(pm.tasks, task)
}

// runTask runs the task and store its result
func (pm *PerformanceMonitor) runTask(ctx context.Context, task Task) error {
	res, err := task.Run(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to run task: %s", task.ID())
	}

	err = pm.setCache(ctx, TaskResult{
		Name:      task.ID(),
		Timestamp: uint64(time.Now().Unix()),
		Result:    res,
	})
	if err != nil {
		return errors.Wrap(err, "failed to set cache")
	}

	return nil
}

// Run adds the tasks to the corn queue and start the scheduler
func (pm *PerformanceMonitor) Run(ctx context.Context) error {
	for _, task := range pm.tasks {
		if _, err := pm.scheduler.CronWithSeconds(task.Cron()).Do(func() error {
			return pm.runTask(ctx, task)
		}); err != nil {
			return errors.Wrapf(err, "failed to schedule the task: %s", task.ID())
		}

		go func(task Task) {
			ok, err := pm.exists(task.ID())
			if err != nil {
				log.Error().Err(err).Msgf("failed to find key %s", task.ID())
			}

			if !ok {
				if err := pm.runTask(ctx, task); err != nil {
					log.Error().Err(err).Msgf("failed to run task: %s", task.ID())
				}
			}
		}(task)

	}

	pm.scheduler.StartAsync()
	return nil
}
