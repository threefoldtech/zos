package provision

import (
	"encoding/json"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules/stubs"
)

// DiskType defines disk type
type DiskType string

const (
	// HDDDiskType for hdd disks
	HDDDiskType DiskType = "HDD"
	// SSDDiskType for ssd disks
	SSDDiskType DiskType = "SSD"
)

// Volume defines a mount point
type Volume struct {
	// Size of the volume in GiB
	Size uint64 `json:"size"`
	// Type of disk underneath the volume
	Type DiskType `json:"type"`
}

// VolumeProvision is entry point to provision a volume
func VolumeProvision(client zbus.Client, reservation Reservation) (interface{}, error) {
	var config Volume
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return nil, err
	}

	storageClient := stubs.NewStorageModuleStub(client)

	return storageClient.CreateFilesystem(config.Size)
}
