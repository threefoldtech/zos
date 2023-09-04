package perf

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

// TaskResult the result test schema
type TaskResult struct {
	TaskName     string      `json:"task_name"`
	RunTimestamp string      `json:"run_timestamp"`
	Result       interface{} `json:"result"`
}

// setCache set result in redis
func (pm *PerformanceMonitor) setCache(ctx context.Context, taskName string, result TaskResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data to JSON")
	}

	_, err = pm.redisConn.Do("SET", taskName, data)
	return err
}

// GetCache get data from redis
func (pm *PerformanceMonitor) GetCache(taskName string) (TaskResult, error) {
	var res TaskResult

	data, err := pm.redisConn.Do("GET", taskName)
	if err != nil {
		return res, errors.Wrap(err, "failed to get the cached result")
	}

	err = json.Unmarshal(data.([]byte), &res)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal data from json")
	}

	return res, nil
}
