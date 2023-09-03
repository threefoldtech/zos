package perf

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

type PerfTest func() (interface{}, error)

// TestLogging simple helper method that ensure scheduler is working
func TestLogging(ctx context.Context) (interface{}, error) {
	log.Info().Msgf("TestLogging: time is: %v", time.Now().Hour())
	return nil, nil
}

// should run every 5 min
// ping 12 point, each 3 times and make average result
// iperf 12 grid node with ipv4
// iperf 12 grid node with ipv6
// uses the iperf binary
func TestNetworkPerformance(ctx context.Context) (interface{}, error) {
	return nil, nil
}

// should run every 6min
// upload/download a 1 MB file to any point
func TestNetworkLoading(ctx context.Context) (interface{}, error) {
	return nil, nil
}

// should run every 6 hours
// measure cpu, mem, disk performance and usage
// uses some tool that do the mentoring
func TestResourcesPerformance(ctx context.Context) (interface{}, error) {
	return nil, nil
}
