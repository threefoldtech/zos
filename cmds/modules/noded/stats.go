package noded

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/client"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	operationTimeout     = 1 * time.Minute
	ReportInterval       = 5 * time.Minute
	ReportMaxElapsedTime = 3 * time.Minute // must be less than report interval
	CyclesToUpdate       = 3
)

func reportStatistics(ctx context.Context, redis string, cl zbus.Client) error {
	stats := stubs.NewStatisticsStub(cl)
	total := stats.Total(ctx)
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "couldn't get environment")
	}
	ctx2, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	oracle := capacity.NewResourceOracle(stubs.NewStorageModuleStub(cl))
	version := stubs.NewVersionMonitorStub(cl).GetVersion(ctx2).String()
	hypervisor, err := oracle.GetHypervisor()
	if err != nil {
		return errors.Wrap(err, "failed to get hypervisors")
	}
	bus, err := rmb.NewClient(redis)
	if err != nil {
		return errors.Wrap(err, "couldn't get an rmb bus instance")
	}
	tc := time.NewTicker(ReportInterval)
	updateCounter := CyclesToUpdate
	extended, err := environment.GetExtended(env.RunningMode)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tc.C:
			if updateCounter == 0 {
				extended, err = environment.GetExtended(env.RunningMode)
				if err != nil {
					log.Error().Err(err).Msg("couldn't get twins to report to")
				}
				updateCounter = CyclesToUpdate
			}
			updateCounter--

			// TODO: .Current should return error
			current := stats.Current(ctx)
			report := client.NodeStatus{
				Current:    current,
				Total:      total,
				ZosVersion: version,
				Hypervisor: hypervisor,
			}
			for _, twin := range extended.Monitor {
				go func(twinID uint32) {
					cl := client.NewProxyClient(twinID, bus)
					if err := sendStatisticsReport(ctx, cl, report); err != nil {
						log.Error().Err(err).Uint32("twin", twinID).Msg("couldn't send report to twin")
					}
				}(twin)
			}
		}
	}
}

func sendStatisticsReport(ctx context.Context, cl *client.ProxyClient, report client.NodeStatus) error {
	ctx2, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	errHandler := func(err error, t time.Duration) {
		if err != nil {
			log.Error().Err(err).Msg("error while sending twin report")
		}
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxInterval = 10 * time.Second
	exp.MaxElapsedTime = ReportMaxElapsedTime
	return backoff.RetryNotify(func() error {
		return cl.ReportStats(ctx2, report)
	}, exp, errHandler)

}
