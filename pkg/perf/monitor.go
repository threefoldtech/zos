package perf

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis"
	"github.com/jasonlvhit/gocron"
	"github.com/pkg/errors"
)

// TaskMethod is the task method signature
type TaskMethod func(ctx context.Context) (interface{}, error)

type Task struct {
	Key            string
	Method         TaskMethod
	Interval       time.Duration // interval in seconds
	ExecutionCount uint64
}

// PerformanceMonitor holds the module data
type PerformanceMonitor struct {
	Scheduler   *gocron.Scheduler
	RedisClient *redis.Client
	Tasks       []Task
}

// NewPerformanceMonitor returns PerformanceMonitor instance
func NewPerformanceMonitor(redisAddr string) *PerformanceMonitor {
	redisClient := redis.NewClient(&redis.Options{
		Network: "unix",
		Addr:    redisAddr,
	})

	scheduler := gocron.NewScheduler()

	return &PerformanceMonitor{
		Scheduler:   scheduler,
		RedisClient: redisClient,
		Tasks:       []Task{},
	}
}

// AddTask a simple helper method to add new tasks
func (pm *PerformanceMonitor) AddTask(taskName string, interval time.Duration, task TaskMethod) {
	pm.Tasks = append(pm.Tasks, Task{
		Key:            taskName,
		Method:         task,
		Interval:       interval,
		ExecutionCount: 0,
	})
}

// InitScheduler adds all the test to the scheduler queue
func (pm *PerformanceMonitor) InitScheduler() {
	pm.AddTask("TestLogging", 5, TestLogging)
}

// RunScheduler adds the tasks to the corn queue and start the scheduler
func (pm *PerformanceMonitor) RunScheduler(ctx context.Context) error {
	for _, task := range pm.Tasks {
		err := pm.Scheduler.Every(uint64(task.Interval)).Seconds().Do(func() error {
			testResult, err := task.Method(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed running test: %s", task.Key)
			}

			task.ExecutionCount++
			err = pm.CacheResult(ctx, task.Key, TestResultData{
				TestName:   task.Key,
				TestNumber: task.ExecutionCount,
				Result:     testResult,
			})
			if err != nil {
				return errors.Wrap(err, "failed setting cache")
			}

			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "failed scheduling the job: %s", task.Key)
		}
	}

	pm.Scheduler.Start()
	return nil
}

// Get handles the request to get data from cache
func (pm *PerformanceMonitor) Get(ctx context.Context, payload []byte) (interface{}, error) {
	var req GetRequest
	err := json.Unmarshal([]byte(payload), &req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal payload: %v", payload)
	}

	return pm.GetCachedResult(req.TestName)
}
