package monitord

import (
	"context"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/net"
	"github.com/threefoldtech/zos/pkg/capacity/dmi"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
)

// NodeAssets is a struct that hold all the node's hardware information
type NodeAssets struct {
	Host              *host.InfoStat       `json:"host"`
	CPUS              []cpu.InfoStat       `json:"cpus"`
	Drives            map[string]*diskStat `json:"drives"`
	NetworkInterfaces []net.InterfaceStat  `json:"interfaces"`
	Memory            *dmi.DMI             `json:"memory"`
}

type diskStat struct {
	Path   string `json:"path"`
	Fstype string `json:"fstype"`
	Total  uint64 `json:"total"`
}

// GetAssets returns node assets
func GetAssets() (NodeAssets, error) {
	nodeAssets := NodeAssets{}

	hostStat, err := host.Info()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.Host = hostStat

	nodeAssets.Drives = make(map[string]*diskStat)

	deviceManager := filesystem.DefaultDeviceManager(context.Background())
	devices, err := deviceManager.Devices(context.Background())
	if err != nil {
		return nodeAssets, err
	}

	for _, dev := range devices {
		diskUsageStat, err := disk.Usage(dev.Path)
		if err != nil {
			return nodeAssets, err
		}
		nodeAssets.Drives[dev.Path] = &diskStat{
			Path:   dev.Path,
			Fstype: diskUsageStat.Fstype,
			Total:  diskUsageStat.Total,
		}
	}

	cpuStat, err := cpu.Info()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.CPUS = cpuStat

	interfStat, err := net.Interfaces()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.NetworkInterfaces = interfStat

	dmi, err := dmi.Decode()
	if err != nil {
		return nodeAssets, err
	}
	nodeAssets.Memory = dmi

	return nodeAssets, nil
}