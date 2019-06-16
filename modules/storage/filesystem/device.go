package filesystem

import (
	"context"
	"encoding/json"
)

type DeviceManager interface {
	Devices(ctx context.Context) ([]Device, error)
}

// Device represents a physical device
type Device struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Path  string `json:"path"`
	Label string `json:"label"`
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
