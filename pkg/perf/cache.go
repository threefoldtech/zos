package perf

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// TestResultData the result test schema
type TestResultData struct {
	TestName   string
	TestNumber uint64
	Result     interface{}
}

// GetRequest the get request struct
type GetRequest struct {
	TestName   string `json:"test_name"`
	TestNumber uint64 `json:"test_number"`
}

// CacheResult set result in redis
func (pm *PerformanceMonitor) CacheResult(ctx context.Context, resultKey string, resultData TestResultData) error {
	data, err := json.Marshal(resultData)
	if err != nil {
		return errors.Wrap(err, "Error marshaling data to JSON")
	}

	return pm.RedisClient.Set(resultKey, data, 10*time.Second).Err()
}

// GetCachedResult get data from redis
func (pm *PerformanceMonitor) GetCachedResult(resultKey string) (TestResultData, error) {
	var res TestResultData

	data, err := pm.RedisClient.Get(resultKey).Result()
	if err != nil {
		return res, errors.Wrap(err, "Failed getting the cached result")
	}

	err = json.Unmarshal([]byte(data), &res)
	if err != nil {
		return res, errors.Wrap(err, "Failed unmarshal data from json")
	}

	return res, nil
}
