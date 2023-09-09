package perf

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
)

const (
	moduleName = "perf"
)

// TaskResult the result test schema
type TaskResult struct {
	TaskName  string      `json:"task_name"`
	Timestamp uint64      `json:"timestamp"`
	Result    interface{} `json:"result"`
}

func generateKey(taskName string) string {
	return fmt.Sprintf("%s.%s", moduleName, taskName)
}

// setCache set result in redis
func (pm *PerformanceMonitor) setCache(ctx context.Context, result TaskResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data to JSON")
	}

	conn := pm.pool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", generateKey(result.TaskName), data)
	return err
}

// Get gets data from redis
func (pm *PerformanceMonitor) Get(taskName string) (TaskResult, error) {
	var res TaskResult

	conn := pm.pool.Get()
	defer conn.Close()

	data, err := conn.Do("GET", generateKey(taskName))
	if err != nil {
		return res, errors.Wrap(err, "failed to get the result")
	}

	err = json.Unmarshal(data.([]byte), &res)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal data from json")
	}

	return res, nil
}

func (pm *PerformanceMonitor) GetAll() ([]TaskResult, error) {
	var res []TaskResult

	conn := pm.pool.Get()
	defer conn.Close()

	var keys []string

	cursor := 0
	for {
		values, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", generateKey("*")))
		if err != nil {
			return nil, err
		}

		_, err = redis.Scan(values, &cursor, &keys)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			value, err := conn.Do("GET", key)
			if err != nil {
				continue
			}

			var result TaskResult

			err = json.Unmarshal(value.([]byte), &result)
			if err != nil {
				return res, errors.Wrap(err, "failed to unmarshal data from json")
			}

			res = append(res, result)
		}

		if cursor == 0 {
			break
		}

	}
	return res, nil
}
