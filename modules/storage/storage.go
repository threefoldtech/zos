package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/storage/filesystem"
)

type storageModule struct {
}

// New create a new storage module service
func New() modules.StorageModule {
	return &storageModule{}
}

/**
Initialize, must be called at least onetime each boot.
What Initialize will do is the following:
 - Try to mount prepared pools (if they are not mounted already)
 - Scan free devices, apply the policy.
 - If new pools were created, the pool is going to be mounted automatically
**/
func (s *storageModule) Initialize(policy modules.StoragePolicy) error {
	log.Info().Msgf("Initializing storage module")

	ctx := context.Background()

	// remount all existing pools
	devices := filesystem.DefaultDeviceManager()
	fs := filesystem.NewBtrfs(devices)

	existingPools, err := fs.List(ctx)
	if err != nil {
		return err
	}

	for _, volume := range existingPools {
		if _, mounted := volume.Mounted(); mounted {
			// volume is aready mounted, skip mounting it again
			continue
		}
		_, err = volume.Mount()
		if err != nil {
			return err
		}
	}

	// list disks
	disks, err := devices.Devices(ctx)
	if err != nil {
		return err
	}

	// collect free disks
	freeDisks := []filesystem.Device{}
	for _, device := range disks {
		if !device.Used() {
			freeDisks = append(freeDisks, device)
		}
	}

	// create new pools if applicable
	newPools := []filesystem.Pool{}
	switch policy.Raid {
	case modules.Single:
		possiblePools := len(freeDisks) / int(policy.Disks)
		for i := 0; i < possiblePools; i++ {
			poolDevices := []string{}
			for j := 0; j < int(policy.Disks); j++ {
				poolDevices = append(poolDevices, freeDisks[i*int(policy.Disks)+j].Path)
			}
			pool, err := fs.Create(ctx, uuid.New().String(), poolDevices, policy.Raid)
			if err != nil {
				return err
			}
			newPools = append(newPools, pool)
		}

	case modules.Raid1:
		return errors.New("not implemented yet")
	case modules.Raid10:
		return errors.New("not implemented yet")
	default:
		return fmt.Errorf("unrecognized storage policy %s", policy.Raid)
	}

	// make sure new pools are mounted
	for _, pool := range newPools {
		if _, mounted := pool.Mounted(); !mounted {
			if _, err = pool.Mount(); err != nil {
				return err
			}
		}
	}

	return nil
}
<<<<<<< Updated upstream
=======

func (s *storageModule) CreateFilesystem(size uint64) (string, error) {
	return "", nil
}

func (s *storageModule) ReleaseFilesystem(path string) error {
	return nil
}
>>>>>>> Stashed changes
