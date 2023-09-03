package perf

import (
	"context"
	"time"

	"github.com/go-redis/redis"
	"github.com/jasonlvhit/gocron"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Task is the task method signature
type Task func(ctx context.Context) (interface{}, error)

// PerformanceMonitor holds the module data
type PerformanceMonitor struct {
	Scheduler       *gocron.Scheduler
	RedisClient     *redis.Client
	Intervals       map[string]time.Duration // interval in seconds
	Tasks           map[string]Task
	ExecutionCounts map[string]uint64
}

// NewPerformanceMonitor returns PerformanceMonitor instance
func NewPerformanceMonitor(redisAddr string) *PerformanceMonitor {
	redisClient := redis.NewClient(&redis.Options{
		Network: "unix",
		Addr:    redisAddr,
	})

	redisClient.Ping()
	log.Info().Msg("redis passed")

	scheduler := gocron.NewScheduler()

	return &PerformanceMonitor{
		Scheduler:       scheduler,
		RedisClient:     redisClient,
		Tasks:           make(map[string]Task),
		Intervals:       make(map[string]time.Duration),
		ExecutionCounts: make(map[string]uint64),
	}
}

// AddTask a simple helper method to add new tasks
func (pm *PerformanceMonitor) AddTask(taskName string, interval time.Duration, task Task) {
	pm.Intervals[taskName] = interval
	pm.Tasks[taskName] = task
	pm.ExecutionCounts[taskName] = 0
}

// InitScheduler adds all the test to the scheduler queue
func (pm *PerformanceMonitor) InitScheduler() {
	pm.AddTask("TestLogging", 5, TestLogging)
}

// RunScheduler adds the tasks to the corn queue and start the scheduler
func (pm *PerformanceMonitor) RunScheduler(ctx context.Context) error {
	for key, task := range pm.Tasks {
		err := pm.Scheduler.Every(uint64(pm.Intervals[key])).Seconds().Do(func() error {
			testResult, err := task(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed running test: %s", key)
			}

			count := pm.ExecutionCounts[key]
			pm.ExecutionCounts[key]++

			err = pm.CacheResult(ctx, key, TestResultData{
				TestName:   key,
				TestNumber: count + 1,
				Result:     testResult,
			})
			if err != nil {
				return errors.Wrap(err, "failed setting cache")
			}

			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "failed scheduling the job: %s", key)
		}
	}

	pm.Scheduler.Start()
	return nil
}
