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
	_ pkg.SystemMonitor = (*SystemMonitor)(nil)
)

// SystemMonitor stream
type SystemMonitor struct {
	Duration time.Duration
}

func (m *SystemMonitor) duration() time.Duration {
	if m.Duration == time.Duration(0) {
		m.Duration = 2 * time.Second
	}

	return m.Duration
}

// Memory starts memory monitor stream
func (m *SystemMonitor) Memory(ctx context.Context) <-chan pkg.VirtualMemoryStat {
	duration := m.duration()
	ch := make(chan pkg.VirtualMemoryStat)
	go func() {
		defer close(ch)

		for {
			select {
			case <-time.After(duration):
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
func (m *SystemMonitor) CPU(ctx context.Context) <-chan pkg.CPUTimesStat {
	duration := m.duration()
	ch := make(chan pkg.CPUTimesStat)
	go func() {
		defer close(ch)

		for {
			select {
			case <-time.After(duration):
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
func (m *SystemMonitor) Disks(ctx context.Context) <-chan pkg.DisksIOCountersStat {
	duration := m.duration()

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
			case <-time.After(duration):
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
