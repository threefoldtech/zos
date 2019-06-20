package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/storage/filesystem"
)

const (
	cacheTarget = "/var/cache"
	cacheLabel  = "zos-cache"
	cacheSize   = 20 * 1024 * 1024 * 1024 // 20GB
)

var (
	diskBase = map[modules.RaidProfile]int{
		modules.Single: 1,
		modules.Raid1:  2,
		modules.Raid10: 4,
	}
)

type storageModule struct {
	volumes []filesystem.Pool
}

// New create a new storage module service
func New() modules.StorageModule {
	return &storageModule{volumes: []filesystem.Pool{}}
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
	defer log.Info().Msgf("Finished initializing storage module")

	ctx := context.Background()

	devices := filesystem.DefaultDeviceManager()
	fs := filesystem.NewBtrfs(devices)

	// remount all existing pools
	log.Info().Msgf("Remounting existing volumes")
	log.Debug().Msgf("Searching for existing volumes")
	existingPools, err := fs.List(ctx)
	if err != nil {
		return err
	}

	for _, volume := range existingPools {
		if _, mounted := volume.Mounted(); mounted {
			log.Debug().Msgf("Volume %s already mounted", volume.Name())
			// volume is aready mounted, skip mounting it again
			continue
		}
		_, err = volume.Mount()
		if err != nil {
			return err
		}
		log.Debug().Msgf("Mounted volume %s", volume.Name())
	}
	s.volumes = append(s.volumes, existingPools...)

	// list disks
	log.Info().Msgf("Finding free disks")
	disks, err := devices.Devices(ctx)
	if err != nil {
		return err
	}

	// collect free disks
	freeDisks := []filesystem.Device{}
	for _, device := range disks {
		if !device.Used() {
			log.Debug().Msgf("Found free device %s", device.Path)
			freeDisks = append(freeDisks, device)
		}
	}

	log.Info().Msgf("Creating new volumes using policy %s", policy.Raid)

	// sanity check for disk amount
	diskBase, exists := diskBase[policy.Raid]
	if !exists {
		return fmt.Errorf("unrecognized storage policy %s", policy.Raid)
	}
	if diskBase%int(policy.Disks) != 0 {
		return fmt.Errorf("invalid amount of disks (%d) for volume for configuration %v", policy.Disks, policy.Raid)
	}

	// create new pools if applicable
	newPools := []filesystem.Pool{}

	possiblePools := len(freeDisks) / int(policy.Disks)
	log.Debug().Msgf("Creating %d new volumes", possiblePools)

	for i := 0; i < possiblePools; i++ {
		log.Debug().Msgf("Creating new volume %d", i)
		poolDevices := []string{}
		for j := 0; j < int(policy.Disks); j++ {
			log.Debug().Msgf("Grabbing device %d: %s for new volume", i*int(policy.Disks)+j, freeDisks[i*int(policy.Disks)+j].Path)
			poolDevices = append(poolDevices, freeDisks[i*int(policy.Disks)+j].Path)
		}
		pool, err := fs.Create(ctx, uuid.New().String(), poolDevices, policy.Raid)
		if err != nil {
			return err
		}
		newPools = append(newPools, pool)
	}

	// make sure new pools are mounted
	log.Info().Msgf("Making sure new volumes are mounted")
	for _, pool := range newPools {
		if _, mounted := pool.Mounted(); !mounted {
			log.Debug().Msgf("Mounting volume %s", pool.Name())
			if _, err = pool.Mount(); err != nil {
				return err
			}
		}
	}

	s.volumes = append(s.volumes, newPools...)

	return s.ensureCache()
}

func (s *storageModule) CreateFilesystem(size uint64) (string, error) {
	log.Info().Msgf("Creating new volume with size %d", size)

	fs, err := s.createFs(size, uuid.New().String())
	if err != nil {
		return "", err
	}
	return fs.Path(), nil
}

func (s *storageModule) ReleaseFilesystem(path string) error {
	log.Info().Msgf("Deleting volume at %v", path)

	for _, volume := range s.volumes {
		filesystems, err := volume.Volumes()
		if err != nil {
			return err
		}
		for _, fs := range filesystems {
			if fs.Path() == path {
				log.Debug().Msgf("Removing filesystem %v in volume %v", fs.Name(), volume.Name())
				return volume.RemoveVolume(fs.Name())
			}
		}
	}

	log.Warn().Msgf("Could not find filesystem %v", path)
	return nil
}

// ensureCache creates a "cache" subvolume and mounts it in /var
func (s *storageModule) ensureCache() error {
	log.Info().Msgf("Setting up cache")

	// TODO: Need to make sure the cache is in an SSD pool

	log.Debug().Msgf("Checking pools for existing cache")

	var cacheFs filesystem.Volume

	// check if we already have a cache
	for _, volume := range s.volumes {
		filesystems, err := volume.Volumes()
		if err != nil {
			return err
		}
		for _, fs := range filesystems {
			if fs.Name() == cacheLabel {
				log.Debug().Msgf("Found existing cache at %v", fs.Path())
				cacheFs = fs
				break
			}
		}
		if cacheFs != nil {
			break
		}
	}

	if cacheFs == nil {
		log.Debug().Msgf("No cache found, try to create new cache")

		fs, err := s.createFs(cacheSize, cacheLabel)
		if err != nil {
			return err
		}
		cacheFs = fs
	}

	log.Debug().Msgf("Mounting cache partition in %s", cacheTarget)
	return filesystem.BindMount(cacheFs, cacheTarget)
}

func (s *storageModule) createFs(size uint64, name string) (filesystem.Volume, error) {
	var filesystem filesystem.Volume
	var err error

	// for now randomly select a volume
	for _, volume := range s.volumes {
		fs, err := volume.AddVolume(name)
		if err != nil {
			log.Error().Msgf("Failed to create new filesystem: %v", err)
			continue
		}
		if err = fs.Limit(size); err != nil {
			log.Error().Msgf("Failed to set size limit on new filesystem: %v", err)
			continue
		}
		filesystem = fs
		break
	}

	return filesystem, err
}
