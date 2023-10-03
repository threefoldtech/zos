package perf

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/threefoldtech/zos/pkg/stubs"
)

// CPUBenchmarkTask defines CPU benchmark task data.
type CPUBenchmarkTask struct {
	taskID   string
	schedule string
}

// CPUBenchmarkResult holds CPU benchmark results with the workloads number during the benchmark.
type CPUBenchmarkResult struct {
	Single    float64 `json:"single"`
	Multi     float64 `json:"multi"`
	Threads   int     `json:"threads"`
	Workloads int     `json:"workloads"`
}

var _ Task = (*CPUBenchmarkTask)(nil)

// NewCPUBenchmarkTask returns a new CPU benchmark task.
func NewCPUBenchmarkTask() CPUBenchmarkTask {
	return CPUBenchmarkTask{
		taskID:   "CPUBenchmark",
		schedule: "0 0 */6 * * *",
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
	cpubenchData := struct {
		Single  float64 `json:"single"`
		Multi   float64 `json:"multi"`
		Threads int     `json:"threads"`
	}{}
	err = json.Unmarshal(cpubenchOut, &cpubenchData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cpubench output: %w", err)
	}
	client := getZbusClient(ctx)
	statistics := stubs.NewStatisticsStub(client)

	workloads, err := statistics.Workloads(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workloads number: %w", err)
	}

	return CPUBenchmarkResult{
		Single:    cpubenchData.Single,
		Multi:     cpubenchData.Multi,
		Threads:   cpubenchData.Threads,
		Workloads: workloads,
	}, nil
}
