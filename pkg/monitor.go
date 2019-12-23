package pkg

//go:generate mkdir -p stubs
//go:generate zbusc -module monitor -version 0.0.1 -name monitor -package stubs github.com/threefoldtech/zos/pkg+Monitor stubs/monitor_stub.go

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// VirtualMemoryStat struct
type VirtualMemoryStat mem.VirtualMemoryStat

// IOCountersStat struct
type IOCountersStat disk.IOCountersStat

//TimesStat struct
type TimesStat struct {
	cpu.TimesStat
	Percent float64
	Time    time.Time
}

func (t *TimesStat) String() string {
	return fmt.Sprintf("CPU: %s Percent: %0.00f (U: %0.00f, S: %0.00f, I: %0.00f)",
		t.CPU, t.Percent, t.User, t.System, t.Idle)
}

// CPUTimesStat alias for []TimesStat required by zbus
type CPUTimesStat []TimesStat

// DisksIOCountersStat alias for map[string]IOCountersStat required by zbus
type DisksIOCountersStat map[string]IOCountersStat

//Monitor interface
type Monitor interface {
	Memory(ctx context.Context) <-chan VirtualMemoryStat
	CPU(ctx context.Context) <-chan CPUTimesStat
	Disks(ctx context.Context) <-chan DisksIOCountersStat
}
