package monitord

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/threefoldtech/zos/pkg"
)

var (
	_ pkg.SystemMonitor = (*systemMonitor)(nil)
)

// systemMonitor stream
type systemMonitor struct {
	duration time.Duration
}

// NewSystemMonitor creates new system of system monitor
func NewSystemMonitor(duration time.Duration) (pkg.SystemMonitor, error) {
	if duration == 0 {
		duration = 2 * time.Second
	}

	return &systemMonitor{duration: duration}, nil
}

// Memory starts memory monitor stream
func (m *systemMonitor) Memory(ctx context.Context) <-chan pkg.VirtualMemoryStat {
	ch := make(chan pkg.VirtualMemoryStat)
	go func() {
		defer close(ch)

		for {
			select {
			case <-time.After(m.duration):
				vm, err := mem.VirtualMemory()
				if err != nil {
					log.Error().Err(err).Msg("failed to read memory status")
					continue
				}

				ch <- pkg.VirtualMemoryStat(*vm)
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// CPU starts cpu monitor stream
func (m *systemMonitor) CPU(ctx context.Context) <-chan pkg.CPUTimesStat {
	ch := make(chan pkg.CPUTimesStat)
	go func() {
		defer close(ch)

		for {
			select {
			case <-time.After(m.duration):
				percents, err := cpu.PercentWithContext(ctx, 0, true)
				if err != nil {
					log.Error().Err(err).Msg("failed to read cpu usage percentage")
					continue
				}
				var result []pkg.TimesStat
				now := time.Now()
				times, err := cpu.Times(true)
				for i, time := range times {
					result = append(result, pkg.TimesStat{
						TimesStat: time,
						Percent:   percents[i],
						Time:      now,
					})
				}
				ch <- result
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// Disks starts disk monitor stream
func (m *systemMonitor) Disks(ctx context.Context) <-chan pkg.DisksIOCountersStat {
	ch := make(chan pkg.DisksIOCountersStat)
	go func() {
		defer close(ch)
		disks, err := disk.PartitionsWithContext(ctx, false)
		if err != nil {
			log.Error().Err(err).Msg("failed to list machine disks")
			return
		}
		var names []string
		for _, device := range disks {
			names = append(names, device.Device)
		}

		for {
			select {
			case <-time.After(m.duration):
				now := time.Now()
				counter, err := disk.IOCountersWithContext(ctx, names...)
				if err != nil {
					log.Error().Err(err).Msg("failed to read IO counter for disks")
				}
				result := make(pkg.DisksIOCountersStat)
				for k, v := range counter {
					result[k] = pkg.IOCountersStat{
						IOCountersStat: v,
						Time:           now,
					}
				}

				ch <- result
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}
