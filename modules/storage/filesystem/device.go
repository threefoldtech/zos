package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
)

type DeviceManager interface {
	Device(ctx context.Context, device string) (Device, error)
	Devices(ctx context.Context) ([]Device, error)
	WithLabel(ctx context.Context, label string) ([]Device, error)
}

// FSType type of filesystem on device
type FSType string

const (
	// BtrfsFSType btrfs filesystem type
	BtrfsFSType FSType = "btrfs"
)

// Device represents a physical device
type Device struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Path       string `json:"path"`
	Label      string `json:"label"`
	Filesystem FSType `json:"fstype"`
}

// Used assumes that the device is used if it has custom label of fstype
func (d *Device) Used() bool {
	return len(d.Label) != 0 || len(d.Filesystem) != 0
}

type lsblkDeviceManager struct{}

// DefaultDeviceManager returns a default device manager implementation
func DefaultDeviceManager() DeviceManager {
	return &lsblkDeviceManager{}
}

// Devices gets available block devices
func (l *lsblkDeviceManager) Devices(ctx context.Context) ([]Device, error) {
	bytes, err := run(ctx, "lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2")
	if err != nil {
		return nil, err
	}

	var devices struct {
		BlockDevices []Device `json:"blockdevices"`
	}

	if err := json.Unmarshal(bytes, &devices); err != nil {
		return nil, err
	}

	return devices.BlockDevices, nil
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
	bytes, err := run(ctx, "lsblk", "--json", "--output-all", "--bytes", "--exclude", "1,2")
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
