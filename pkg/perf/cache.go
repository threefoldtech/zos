package perf

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
)

const (
	moduleName = "perf"
)

var (
	ErrResultNotFound = errors.New("result not found")
)

// GeneratePerfKey is helper method to add moduleName as prefix for the taskName
func GeneratePerfKey(taskName string) string {
	return fmt.Sprintf("%s.%s", moduleName, taskName)
}

// exists check if a key exists
func (pm *PerformanceMonitor) exists(key string) (bool, error) {
	conn := pm.pool.Get()
	defer conn.Close()

	ok, err := redis.Bool(conn.Do("EXISTS", GeneratePerfKey(key)))
	if err != nil {
		return false, errors.Wrapf(err, "error checking if key %s exists", GeneratePerfKey(key))
	}
	return ok, nil
}

// setCache set result in redis
func (pm *PerformanceMonitor) setCache(_ context.Context, result pkg.TaskResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data to JSON")
	}

	conn := pm.pool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", GeneratePerfKey(result.Name), data)
	return err
}

func (pm *PerformanceMonitor) getTaskResult(conn redis.Conn, key string, result interface{}) error {
	data, err := conn.Do("GET", GeneratePerfKey(key))
	if err != nil {
		return errors.Wrap(err, "failed to get the result")
	}

	if data == nil {
		return ErrResultNotFound
	}

	err = json.Unmarshal(data.([]byte), result)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal data from json")
	}

	return nil
}

func (pm *PerformanceMonitor) GetIperfTaskResult() (pkg.IperfTaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()

	var res pkg.IperfTaskResult
	err := pm.getTaskResult(conn, pkg.IperfTaskName, &res)
	return res, err
}

func (pm *PerformanceMonitor) GetHealthTaskResult() (pkg.HealthTaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()

	var res pkg.HealthTaskResult
	err := pm.getTaskResult(conn, pkg.HealthCheckTaskName, &res)
	return res, err
}

func (pm *PerformanceMonitor) GetPublicIpTaskResult() (pkg.PublicIpTaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()

	var res pkg.PublicIpTaskResult
	err := pm.getTaskResult(conn, pkg.PublicIpTaskName, &res)
	return res, err
}

func (pm *PerformanceMonitor) GetCpuBenchTaskResult() (pkg.CpuBenchTaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()

	var res pkg.CpuBenchTaskResult
	err := pm.getTaskResult(conn, pkg.CpuBenchmarkTaskName, &res)
	return res, err
}

func (pm *PerformanceMonitor) GetAllTaskResult() (pkg.AllTaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()

	var results pkg.AllTaskResult

	var cpu pkg.CpuBenchTaskResult
	if err := pm.getTaskResult(conn, pkg.CpuBenchmarkTaskName, &cpu); err != nil {
		return pkg.AllTaskResult{}, fmt.Errorf("failed to get health result: %w", err)
	}
	results.CpuBenchmark = cpu

	var health pkg.HealthTaskResult
	if err := pm.getTaskResult(conn, pkg.HealthCheckTaskName, &health); err != nil {
		return pkg.AllTaskResult{}, fmt.Errorf("failed to get health result: %w", err)
	}
	results.HealthCheck = health

	var iperf pkg.IperfTaskResult
	if err := pm.getTaskResult(conn, pkg.IperfTaskName, &iperf); err != nil {
		return pkg.AllTaskResult{}, fmt.Errorf("failed to get iperf result: %w", err)
	}
	results.Iperf = iperf

	var pIp pkg.PublicIpTaskResult
	if err := pm.getTaskResult(conn, pkg.PublicIpTaskName, &pIp); err != nil {
		return pkg.AllTaskResult{}, fmt.Errorf("failed to get public ip result: %w", err)
	}
	results.PublicIp = pIp

	return results, nil
}

// DEPRECATED

// get directly gets result for some key
func get(conn redis.Conn, key string) (pkg.TaskResult, error) {
	var res pkg.TaskResult

	data, err := conn.Do("GET", key)
	if err != nil {
		return res, errors.Wrap(err, "failed to get the result")
	}

	if data == nil {
		return res, ErrResultNotFound
	}

	err = json.Unmarshal(data.([]byte), &res)
	if err != nil {
		return res, errors.Wrap(err, "failed to unmarshal data from json")
	}

	return res, nil
}

// Get gets data from redis
func (pm *PerformanceMonitor) Get(taskName string) (pkg.TaskResult, error) {
	conn := pm.pool.Get()
	defer conn.Close()
	return get(conn, GeneratePerfKey(taskName))
}

// GetAll gets the results for all the tests with moduleName as prefix
func (pm *PerformanceMonitor) GetAll() ([]pkg.TaskResult, error) {
	var res []pkg.TaskResult

	conn := pm.pool.Get()
	defer conn.Close()

	var keys []string

	cursor := 0
	for {
		values, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", GeneratePerfKey("*")))
		if err != nil {
			return nil, err
		}

		_, err = redis.Scan(values, &cursor, &keys)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			result, err := get(conn, key)
			if err != nil {
				continue
			}
			res = append(res, result)
		}

		if cursor == 0 {
			break
		}

	}
	return res, nil
}
