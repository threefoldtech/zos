package perf

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/rs/zerolog/log"
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
	_, err := exec.LookPath("iperf")
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "iperf", fmt.Sprintf("-c %s -p %d -b %s", t.ClientIP, iperf.IperfPort, t.Bandwidth))
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		log.Error().Err(err).Msgf("failed to run iperf tcp task: %s", stderr.String())
	}

	return out.String(), nil
}
