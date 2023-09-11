package perf

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/threefoldtech/zos/pkg/network/iperf"
)

// TCPTask is the task for iperf tcp tests
type TCPTask struct {
	TaskID    string
	Schedule  string
	Bandwidth string
	ClientIP  string
}

// ID returns the ID of the tcp task
func (t *TCPTask) ID() string {
	return t.TaskID
}

// Cron returns the schedule for the tcp task
func (t *TCPTask) Cron() string {
	return t.Schedule
}

// Run runs the tcp test and returns the result
func (t *TCPTask) Run(ctx context.Context) (interface{}, error) {
	output, err := exec.CommandContext(ctx, fmt.Sprintf("iperf3 -c %s -p %d -b %s", t.ClientIP, iperf.IperfPort, t.Bandwidth)).Output()
	if err != nil {
		return nil, err
	}

	return output, nil
}
