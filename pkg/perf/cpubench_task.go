package perf

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/threefoldtech/zos/pkg/stubs"
)

const cpuBenchmarkTaskID = "CPUBenchmark"
const cpuBenchmarkCronSchedule = "0 0 */6 * * *"

// CPUBenchmarkTask defines CPU benchmark task data.
type CPUBenchmarkTask struct {
	// taskID is a unique string ID for the task.
	taskID string
	// schedule is a 6 field cron schedule (unlike unix cron).
	schedule string
}

// CPUBenchmarkResult holds CPU benchmark results with the workloads number during the benchmark.
type CPUBenchmarkResult struct {
	SingleThreaded float64 `json:"single"`
	MultiThreaded  float64 `json:"multi"`
	Threads        int     `json:"threads"`
	Workloads      int     `json:"workloads"`
}

var _ Task = (*CPUBenchmarkTask)(nil)

// NewCPUBenchmarkTask returns a new CPU benchmark task.
func NewCPUBenchmarkTask() CPUBenchmarkTask {
	return CPUBenchmarkTask{
		taskID:   cpuBenchmarkTaskID,
		schedule: cpuBenchmarkCronSchedule,
	}
}

// ID returns task ID.
func (c *CPUBenchmarkTask) ID() string {
	return c.taskID
}

// Cron returns task cron schedule.
func (c *CPUBenchmarkTask) Cron() string {
	return c.schedule
}

// Run executes the CPU benchmark.
func (c *CPUBenchmarkTask) Run(ctx context.Context) (interface{}, error) {
	cpubenchOut, err := exec.CommandContext(ctx, "cpubench", "-j").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute cpubench command: %w", err)
	}
	cpuBenchmarkResult := CPUBenchmarkResult{}
	err = json.Unmarshal(cpubenchOut, &cpuBenchmarkResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpubench output: %w", err)
	}
	client := GetZbusClient(ctx)
	statistics := stubs.NewStatisticsStub(client)

	workloads, err := statistics.Workloads(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workloads number: %w", err)
	}

	cpuBenchmarkResult.Workloads = workloads
	return cpuBenchmarkResult, nil
}
