package perf

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/threefoldtech/zos/pkg/network/iperf"
)

// UDPTask is the task for iperf udp tests
type UDPTask struct {
	TaskID    string
	Schedule  string
	Bandwidth string
	ClientIP  string
}

// ID returns the ID of the udp task
func (t *UDPTask) ID() string {
	return t.TaskID
}

// Cron returns the schedule for the udp task
func (t *UDPTask) Cron() string {
	return t.Schedule
}

// Run runs the udp test and returns the result
func (t *UDPTask) Run(ctx context.Context) (interface{}, error) {
	output, err := exec.CommandContext(ctx, fmt.Sprintf("iperf3 -c %s -p %d -b %s -u", t.ClientIP, iperf.IperfPort, t.Bandwidth)).Output()
	if err != nil {
		return nil, err
	}

	return output, nil
}
