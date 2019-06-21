package filesystem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	log "github.com/rs/zerolog/log"
)

type DeviceManager interface {
	Device(ctx context.Context, device string) (Device, error)
	Devices(ctx context.Context) ([]Device, error)
	WithLabel(ctx context.Context, label string) ([]Device, error)
	PoolType(ctx context.Context, pool Pool) (DeviceType, error)
}

// FSType type of filesystem on device
type FSType string

const (
	// BtrfsFSType btrfs filesystem type
	BtrfsFSType FSType = "btrfs"
)

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType string

// Known device types
const (
	SSDDevice = "SSD"
	HDDDevice = "HDD"
)

// Device represents a physical device
type Device struct {
	Type       string   `json:"type"`
	Path       string   `json:"name"`
	Label      string   `json:"label"`
	Filesystem FSType   `json:"fstype"`
	Children   []Device `json:"children"`
	DiskType   DeviceType
	ReadTime   uint64
}

// Used assumes that the device is used if it has custom label of fstype
func (d *Device) Used() bool {
	return len(d.Label) != 0 || len(d.Filesystem) != 0 || len(d.Children) > 0
}

type lsblkDeviceManager struct{}

// DefaultDeviceManager returns a default device manager implementation
func DefaultDeviceManager() DeviceManager {
	return &lsblkDeviceManager{}
}

// Devices gets available block devices
func (l *lsblkDeviceManager) Devices(ctx context.Context) ([]Device, error) {
	bytes, err := run(ctx, "lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2,11", "--path")
	if err != nil {
		return nil, err
	}

	var devices struct {
		BlockDevices []Device `json:"blockdevices"`
	}

	if err := json.Unmarshal(bytes, &devices); err != nil {
		return nil, err
	}

	return setDeviceTypes(flattenDevices(devices.BlockDevices)), nil
}

func (l *lsblkDeviceManager) WithLabel(ctx context.Context, label string) ([]Device, error) {
	devices, err := l.Devices(ctx)
	if err != nil {
		return nil, err
	}
	var filtered []Device

	for _, device := range devices {
		if device.Label == label {
			filtered = append(filtered, device)
		}
	}

	return filtered, nil
}

func (l *lsblkDeviceManager) Device(ctx context.Context, path string) (device Device, err error) {
	bytes, err := run(ctx, "lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2,11", "--path", path)
	if err != nil {
		return device, err
	}

	var devices struct {
		BlockDevices []Device `json:"blockdevices"`
	}

	if err := json.Unmarshal(bytes, &devices); err != nil {
		return device, err
	}

	if len(devices.BlockDevices) != 1 {
		return device, fmt.Errorf("device not found")
	}

	return devices.BlockDevices[0], nil
}

func (l *lsblkDeviceManager) PoolType(ctx context.Context, pool Pool) (DeviceType, error) {
	devices, err := l.Devices(ctx)
	if err != nil {
		return "", err
	}

	poolDevices := []Device{}

	for _, d := range devices {
		if d.Label == pool.Name() {
			poolDevices = append(poolDevices, d)
		}
	}

	if len(poolDevices) == 0 {
		return "", errors.New("Pool has no known devices")
	}

	// assume homogenous pools for now
	return poolDevices[0].DiskType, nil
}

func flattenDevices(devices []Device) []Device {
	list := []Device{}
	for _, d := range devices {
		list = append(list, d)
		if d.Children != nil {
			list = append(list, flattenDevices(d.Children)...)
		}
	}
	return list
}

func setDeviceTypes(devices []Device) []Device {
	list := []Device{}
	for _, d := range devices {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		typ, rt, err := seektime(ctx, d.Path)
		if err != nil {
			// don't include errored devices in the result
			log.Error().Msgf("Failed to get disk read time: %v", err)
			continue
		}
		d.DiskType = deviceTypeFromString(typ)
		d.ReadTime = rt
		list = append(list, d)
	}

	return list
}

func deviceTypeFromString(typ string) DeviceType {
	switch typ {
	case string(SSDDevice):
		return SSDDevice
	case string(HDDDevice):
		return HDDDevice
	default:
		// if we have an error or unrecognized type, set type to HDD
		return HDDDevice
	}
}

// ByReadTime implements sort.Interface for []Device based on the ReadTime field
type ByReadTime []Device

func (a ByReadTime) Len() int           { return len(a) }
func (a ByReadTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByReadTime) Less(i, j int) bool { return a[i].ReadTime < a[j].ReadTime }
