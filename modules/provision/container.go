package provision

import (
	"fmt"

	"github.com/threefoldtech/zbus"
)

// Network struct
type Network struct {
	NetwokID string
}

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
	// Where to attach the volume in the container
	Mountpoint string `json:"mountpoint"`
}

//Container creation info
type Container struct {
	// URL of the flist
	FList string `json:"flist"`
	// Env env variables to container in format
	Env map[string]string `json:"env"`
	// Entrypoint the process to start inside the container
	Entrypoint string `json:"entrypoint"`
	// Interactivity enable Core X as PID 1 on the container
	Interactive bool `json:"interactive"`
	// Mounts extra mounts in the container
	Volumes []Volume `json:"volumes"`
	// Network network info for container
	Network Network `json:"network"`
}

// ContainerProvision is entry point to container reservation
func ContainerProvision(client zbus.Client, reservation Reservation) error {
	return fmt.Errorf("not implemented")
}
