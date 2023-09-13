package perf

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
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
	_, err := exec.LookPath("iperf")
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "iperf", fmt.Sprintf("-c %s -p %d -b %s -u", t.ClientIP, iperf.IperfPort, t.Bandwidth))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		log.Error().Err(err).Msgf("failed to run iperf udp task: %s", stderr.String())
	}

	return out.String(), nil
}
