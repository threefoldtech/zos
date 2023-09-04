package perf

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/gomodule/redigo/redis"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/utils"
)

// PerformanceMonitor holds the module data
type PerformanceMonitor struct {
	scheduler *gocron.Scheduler
	redisConn redis.Conn
	tasks     []Task
}

// NewPerformanceMonitor returns PerformanceMonitor instance
func NewPerformanceMonitor(redisAddr string) (*PerformanceMonitor, error) {
	redisPool, err := utils.NewRedisPool(redisAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating new redis pool")
	}

	redisConn := redisPool.Get()
	scheduler := gocron.NewScheduler(time.UTC)

	return &PerformanceMonitor{
		scheduler: scheduler,
		redisConn: redisConn,
		tasks:     []Task{},
	}, nil
}

// AddTask a simple helper method to add new tasks
func (pm *PerformanceMonitor) AddTask(task Task) {
	pm.tasks = append(pm.tasks, task)
}

// Run adds the tasks to the corn queue and start the scheduler
func (pm *PerformanceMonitor) Run(ctx context.Context) error {
	for _, task := range pm.tasks {
		_, err := pm.scheduler.CronWithSeconds(task.Cron()).Do(func() error {
			res, err := task.Run(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to run task: %s", task.ID())
			}

			err = pm.setCache(ctx, task.ID(), TaskResult{
				TaskName:     task.ID(),
				RunTimestamp: time.Now().Format("2006-01-02 15:04:05"),
				Result:       res,
			})
			if err != nil {
				return errors.Wrap(err, "failed to set cache")
			}

			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "failed to schedule the task: %s", task.ID())
		}
	}

	pm.scheduler.StartAsync()
	return nil
}

// Close closes the redis connection
func (pm *PerformanceMonitor) Close() {
	pm.redisConn.Close()
}
