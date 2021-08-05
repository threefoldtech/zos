package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/disk"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
)

const (
	// CacheTarget is the path where the cache disk is mounted
	CacheTarget = "/var/cache"
	// cacheLabel is the name of the cache
	cacheLabel = "zos-cache"
	gib        = 1024 * 1024 * 1024
	cacheSize  = 100 * gib
)

// Module implements functionality for pkg.StorageModule
type Module struct {
	devices filesystem.DeviceManager

	ssds []filesystem.Pool
	hdds []filesystem.Pool

	brokenPools   []pkg.BrokenPool
	brokenDevices []pkg.BrokenDevice

	totalSSD uint64
	totalHDD uint64

	mu sync.RWMutex
}

// New create a new storage module service
func New() (*Module, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	m := filesystem.DefaultDeviceManager(ctx)

	s := &Module{
		ssds:          []filesystem.Pool{},
		devices:       m,
		brokenDevices: []pkg.BrokenDevice{},
	}

	// go for a simple linear setup right now
	err := s.initialize()

	if err == nil {
		log.Info().Msgf("Finished initializing storage module")
	}

	return s, err
}

// Total gives the total amount of storage available for a device type
func (s *Module) Total(kind pkg.DeviceType) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch kind {
	case zos.SSDDevice:
		return s.totalSSD, nil
	case zos.HDDDevice:
		return s.totalHDD, nil
	default:
		return 0, fmt.Errorf("kind %+v unknown", kind)
	}
}

// BrokenPools lists the broken storage pools that have been detected
func (s *Module) BrokenPools() []pkg.BrokenPool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brokenPools
}

// BrokenDevices lists the broken devices that have been detected
func (s *Module) BrokenDevices() []pkg.BrokenDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.brokenDevices
}

func (s *Module) dump() {
	log.Debug().Int("volumes", len(s.ssds)).Msg("dumping volumes")

	for _, pool := range s.ssds {
		path, err := pool.Mounted()
		if err == nil {
			log.Debug().Msgf("pool %s is mounted at: %s", pool.Name(), path)
		}
		device := pool.Device()
		log.Debug().Str("path", device.Path()).Str("label", pool.Name()).Str("type", string(device.Type())).Send()
	}

}

/**
initialize, must be called at least onetime each boot.
What Initialize will do is the following:
 - Try to mount prepared pools (if they are not mounted already)
 - Scan free devices, apply the policy.
 - If new pools were created, the pool is going to be mounted automatically
**/
func (s *Module) initialize() error {
	// lock for the entire initialization method, so other code which relies
	// on this observes this as an atomic operation
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Info().Msgf("Initializing storage module")

	// Make sure we finish in 1 minute
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()

	devices, err := s.devices.Devices(ctx)
	if err != nil {
		return err
	}

	for _, device := range devices {
		pool, err := filesystem.NewBtrfsPool(device)
		if err != nil {
			log.Error().Err(err).Str("device", device.Path()).Msg("failed to create pool on device")
			s.brokenDevices = append(s.brokenDevices, pkg.BrokenDevice{Path: device.Path(), Err: err})
			continue
		}

		_, err = pool.Mount()
		if err != nil {
			s.brokenPools = append(s.brokenPools, pkg.BrokenPool{Label: pool.Name(), Err: err})
			continue
		}
		usage, err := pool.Usage()
		if err != nil {
			log.Error().Err(err).Str("pool", pool.Name()).Str("device", device.Path()).Msg("failed to get usage of pool")
		}

		switch device.Type() {
		case zos.SSDDevice:
			s.totalSSD += usage.Size
			s.ssds = append(s.ssds, pool)
		case zos.HDDDevice:
			s.totalHDD += usage.Size
			s.hdds = append(s.hdds, pool)
		}
	}

	log.Info().
		Int("ssd-pools", len(s.ssds)).
		Int("hdd-pools", len(s.hdds)).
		Int("broken-pools", len(s.brokenPools)).
		Int("borken-devices", len(s.brokenDevices)).
		Msg("pool creations completed")

	s.dump()

	// just in case
	if err := filesystem.Partprobe(ctx); err != nil {
		return err
	}

	if err := s.ensureCache(); err != nil {
		log.Error().Err(err).Msg("Error ensuring cache")
		return err
	}

	hyperVisor, err := capacity.NewResourceOracle(nil).GetHypervisor()
	if err == nil {
		// Disable disk shutdown when running in a VM
		if len(hyperVisor) > 0 {
			return nil
		}
	}

	if err := s.shutdownUnusedPools(); err != nil {
		log.Error().Err(err).Msg("Error shutting down unused pools")
	}

	s.periodicallyCheckDiskShutdown()

	return nil
}

func (s *Module) shutdownUnusedPools() error {
	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err == nil {
			volumes, err := pool.Volumes()
			if err != nil {
				log.Error().Err(err).Msgf("Failed to retrieve subvolumes on pool %s", pool.Name())
				return err
			}
			log.Debug().Msgf("Pool %s has: %d subvolumes", pool.Name(), len(volumes))

			if len(volumes) > 0 {
				continue
			}
			err = pool.UnMount()
			if err != nil {
				log.Error().Err(err).Msgf("Failed to unmount volume %s", pool.Name())
				return err
			}
		}
		if err := pool.Shutdown(); err != nil {
			log.Error().Err(err).Msgf("Error shutting down pool %s", pool.Name())
		}
	}
	return nil
}

// UpdateFilesystem updates filesystem size
func (s *Module) UpdateFilesystem(name string, size gridtypes.Unit) (pkg.Filesystem, error) {
	_, volume, fs, err := s.path(name)
	if err != nil {
		return pkg.Filesystem{}, err
	}

	if err := volume.Limit(uint64(size)); err != nil {
		return fs, err
	}

	fs.Usage.Size = size
	return fs, nil
}

// CreateFilesystem with the given size in a storage pool.
func (s *Module) CreateFilesystem(name string, size gridtypes.Unit) (pkg.Filesystem, error) {
	log.Info().Msgf("Creating new volume with size %d", size)
	if strings.HasPrefix(name, "zdb") {
		return pkg.Filesystem{}, fmt.Errorf("invalid volume name. zdb prefix is reserved")
	}

	fs, err := s.createSubvolWithQuota(size, name)
	if err != nil {
		return pkg.Filesystem{}, err
	}

	usage, err := fs.Usage()
	if err != nil {
		return pkg.Filesystem{}, err
	}

	return pkg.Filesystem{
		ID:     fs.ID(),
		FsType: fs.FsType(),
		Name:   fs.Name(),
		Path:   fs.Path(),
		Usage: pkg.Usage{
			Size: gridtypes.Unit(usage.Size),
			Used: gridtypes.Unit(usage.Used),
		},
	}, nil
}

// ReleaseFilesystem with the given name, this will unmount and then delete
// the filesystem. After this call, the caller must not perform any more actions
// on this filesystem
func (s *Module) ReleaseFilesystem(name string) error {
	log.Info().Msgf("Deleting volume %v", name)

	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err != nil {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return err
		}
		for _, vol := range volumes {
			if vol.Name() != name {
				continue
			}
			log.Debug().Msgf("Removing filesystem %v in volume %v", vol.Name(), pool.Name())
			err = pool.RemoveVolume(vol.Name())
			if err != nil {
				log.Err(err).Msgf("Error removing volume %s", vol.Name())
				return err
			}
			// if there is only 1 volume, unmount and shutdown pool
			if len(volumes) == 1 {
				err = pool.UnMount()
				if err != nil {
					log.Err(err).Msgf("Error unmounting pool %s", pool.Name())
					return err
				}
				err = pool.Shutdown()
				if err != nil {
					log.Err(err).Msgf("Error shutting down pool %s", pool.Name())
					return err
				}
			}

			return nil
		}
	}

	log.Warn().Msgf("Could not find filesystem %v", name)
	return nil
}

// ListFilesystems return all the filesystem managed by storeaged present on the nodes
func (s *Module) ListFilesystems() ([]pkg.Filesystem, error) {
	fss := make([]pkg.Filesystem, 0, 10)

	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err != nil {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return nil, err
		}
		for _, v := range volumes {
			// Do not return "special" volumes here
			// instead the GetCacheFS and GetVdiskFS to access them
			if v.Name() == cacheLabel ||
				v.Name() == vdiskVolumeName {
				continue
			}

			usage, err := v.Usage()
			if err != nil {
				return nil, err
			}

			fss = append(fss, pkg.Filesystem{
				ID:     v.ID(),
				FsType: v.FsType(),
				Name:   v.Name(),
				Path:   v.Path(),
				Usage: pkg.Usage{
					Size: gridtypes.Unit(usage.Size),
					Used: gridtypes.Unit(usage.Used),
				},
			})
		}
	}

	return fss, nil
}

// Path return the path of the mountpoint of the named filesystem
// if no volume with name exists, an empty path and an error is returned
func (s *Module) Path(name string) (pkg.Filesystem, error) {
	_, _, fs, err := s.path(name)
	return fs, err
}

// Path return the path of the mountpoint of the named filesystem
// if no volume with name exists, an empty path and an error is returned
func (s *Module) path(name string) (filesystem.Pool, filesystem.Volume, pkg.Filesystem, error) {
	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err != nil {
			continue
		}
		filesystems, err := pool.Volumes()
		if err != nil {
			return nil, nil, pkg.Filesystem{}, err
		}
		for _, fs := range filesystems {
			if fs.Name() == name {
				usage, err := fs.Usage()
				if err != nil {
					return nil, nil, pkg.Filesystem{}, err
				}

				return pool, fs, pkg.Filesystem{
					ID:     fs.ID(),
					FsType: fs.FsType(),
					Name:   fs.Name(),
					Path:   fs.Path(),
					Usage: pkg.Usage{
						Size: gridtypes.Unit(usage.Size),
						Used: gridtypes.Unit(usage.Used),
					},
				}, nil
			}
		}
	}

	return nil, nil, pkg.Filesystem{}, errors.Wrapf(os.ErrNotExist, "subvolume '%s' not found", name)
}

// VDiskFindCandidate find a suitbale location for creating a vdisk of the given size
func (s *Module) VDiskFindCandidate(size gridtypes.Unit) (path string, err error) {
	candidates, err := s.findCandidates(size)
	if err != nil {
		return path, err
	}
	// does anyone have a vdisk subvol
	for _, candidate := range candidates {
		volumes, err := candidate.Pool.Volumes()
		if err != nil {
			log.Error().Str("pool", candidate.Pool.Path()).Err(err).Msg("failed to list pool volumes")
			continue
		}
		for _, volume := range volumes {
			if volume.Name() != vdiskVolumeName {
				continue
			}

			return volume.Path(), nil
		}
	}
	// none has a vdiks subvolume, we need to
	// create one.
	candidate := candidates[0]
	volume, err := candidate.Pool.AddVolume(vdiskVolumeName)
	if err != nil {
		return path, errors.Wrap(err, "failed to create vdisk pool")
	}

	return volume.Path(), nil
}

// VDiskPools return a list of all vdisk pools
func (s *Module) VDiskPools() ([]string, error) {
	var paths []string
	for _, pool := range s.ssds {
		if pool.Type() != zos.SSDDevice {
			continue
		}

		if _, err := pool.Mounted(); err != nil {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list pool '%s' volumes", pool.Path())
		}

		for _, volume := range volumes {
			if volume.Name() == vdiskVolumeName {
				paths = append(paths, volume.Path())
			}
		}
	}

	return paths, nil
}

// GetCacheFS return the special filesystem used by 0-OS to store internal state and flist cache
func (s *Module) GetCacheFS() (pkg.Filesystem, error) {
	return s.Path(cacheLabel)
}

// ensureCache creates a "cache" subvolume and mounts it in /var
func (s *Module) ensureCache() error {
	log.Info().Msgf("Setting up cache")

	log.Debug().Msgf("Checking pools for existing cache")

	var cacheFs filesystem.Volume

	// check if cache volume available
	for _, pool := range s.ssds {
		if _, err := pool.Mounted(); err != nil {
			continue
		}

		filesystems, err := pool.Volumes()
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

		log.Debug().Msgf("Trying to create new cache on SSD")
		fs, err := s.createSubvolWithQuota(cacheSize, cacheLabel)

		if err != nil {
			log.Warn().Err(err).Msg("failed to create new cache on SSD")
		} else {
			cacheFs = fs
		}
	}

	if cacheFs == nil {
		log.Warn().Msg("failed to create persisted cache disk. Running on limited cache")

		// set limited cache flag
		if err := app.SetFlag("limited-cache"); err != nil {
			return err
		}

		// when everything failed, mount the Tmpfs
		return syscall.Mount("", "/var/cache", "tmpfs", 0, "size=500M")
	}

	log.Info().Msgf("set cache quota to %d GiB", cacheSize/gib)
	if err := cacheFs.Limit(cacheSize); err != nil {
		log.Error().Err(err).Msg("failed to set cache quota")
	}

	if !filesystem.IsMountPoint(CacheTarget) {
		log.Debug().Msgf("Mounting cache partition in %s", CacheTarget)
		return filesystem.BindMount(cacheFs, CacheTarget)
	}

	log.Debug().Msgf("Cache partition already mounted in %s", CacheTarget)
	return nil
}

// createSubvolWithQuota creates a subvolume with the given name and limits it to the given size
// if the requested disk type does not have a storage pool with enough free size available, an error is returned
// this methods does set a quota limit equal to size on the created volume
func (s *Module) createSubvolWithQuota(size gridtypes.Unit, name string) (filesystem.Volume, error) {
	volume, err := s.createSubvol(size, name)
	if err != nil {
		return nil, err
	}

	if err = volume.Limit(uint64(size)); err != nil {
		log.Error().Err(err).Str("volume", volume.Path()).Msg("failed to set volume size limit")
		return nil, err
	}

	return volume, nil
}

// createSubvol creates a subvolume with the given name
// if the requested disk type does not have a storage pool with enough free size available, an error is returned
// this method does not set any quota on the subvolume, for this uses createSubvolWithQuota
func (s *Module) createSubvol(size gridtypes.Unit, name string) (filesystem.Volume, error) {
	var err error

	// Look for candidates in mounted pools first
	candidates, err := s.findCandidates(size)
	if err != nil {
		log.Error().Err(err).Msgf("failed to search candidates on mounted pools")
		return nil, err
	}

	var volume filesystem.Volume
	for _, candidate := range candidates {
		volume, err = candidate.Pool.AddVolume(name)
		if err != nil {
			log.Error().Err(err).Str("pool", candidate.Pool.Name()).Msg("failed to create new filesystem")
			continue
		}

		return volume, nil
	}

	return nil, fmt.Errorf("failed to create subvolume, logs might have more information")
}

type candidate struct {
	Pool      filesystem.Pool
	Available uint64
}

func (s *Module) findCandidates(size gridtypes.Unit) ([]candidate, error) {

	// Look for candidates in mounted pools first
	candidates, err := s.checkForCandidates(size, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search candidate on mounted pools")
	}

	log.Debug().Msgf("found %d candidates in mounted pools", len(candidates))

	// If no candidates or found in mounted pools, we check the unmounted pools and get the first one that fits
	if len(candidates) == 0 {
		log.Debug().Msg("Checking unmounted pools")
		candidates, err = s.checkForCandidates(size, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search candidates on unmounted pools")
		}

		log.Debug().Msgf("found %d candidates in unmounted pools", len(candidates))
	}

	if len(candidates) == 0 {
		return nil, pkg.ErrNotEnoughSpace{DeviceType: zos.SSDDevice}
	}

	return candidates, nil
}

func (s *Module) checkForCandidates(size gridtypes.Unit, mounted bool) ([]candidate, error) {
	var candidates []candidate
	for _, pool := range s.ssds {
		_, err := pool.Mounted()
		poolIsMounted := err == nil
		if mounted != poolIsMounted {
			continue
		}

		log.Debug().Msgf("checking pool %s for space", pool.Name())

		if !poolIsMounted && !mounted {
			log.Debug().Msgf("Mounting pool %s...", pool.Name())
			// if the pool is not mounted, and we are looking for not mounted pools, mount it first
			_, err := pool.Mount()
			if err != nil {
				log.Error().Err(err).Msgf("failed to mount pool %s", pool.Name())
				return nil, err
			}
		}

		usage, err := pool.Usage()
		if err != nil {
			log.Error().Msgf("Failed to get current volume usage: %v", err)
			continue
		}

		reserved, err := pool.Reserved()
		if err != nil {
			log.Error().Err(err).Msgf("failed to get size of pool %s", pool.Name())
			continue
		}

		log.Debug().
			Uint64("max size", usage.Size).
			Uint64("reserved", reserved).
			Uint64("new size", reserved+uint64(size)).
			Msgf("usage of pool %s", pool.Name())
		// Make sure adding this filesystem would not bring us over the disk limit
		if reserved+uint64(size) > usage.Size {
			log.Info().Msgf("Disk does not have enough space left to hold filesystem")

			if !poolIsMounted && !mounted {
				log.Info().Msgf("Previously unmounted pool shutting down again..")
				err = pool.UnMount()
				if err != nil {
					log.Error().Err(err).Msgf("failed to unmount pool %s", pool.Name())
					return nil, err
				}
				err = pool.Shutdown()
				if err != nil {
					log.Error().Err(err).Msgf("failed to shutdown pool %s", pool.Name())
					return nil, err
				}
			}
			continue
		}

		candidates = append(candidates, candidate{
			Pool:      pool,
			Available: usage.Size - (reserved + uint64(size)),
		})

		// if we are looking for not mounted pools, break here
		if !mounted {
			return candidates, nil
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		// reverse sorting so most available is at beginning
		return candidates[i].Available > candidates[j].Available
	})

	return candidates, nil
}

// Monitor implements monitor method
func (s *Module) Monitor(ctx context.Context) <-chan pkg.PoolsStats {
	ch := make(chan pkg.PoolsStats)
	values := make(pkg.PoolsStats)
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}

			for _, pool := range s.ssds {
				if _, err := pool.Mounted(); err != nil {
					continue
				}
				devices, err := s.devices.ByLabel(ctx, pool.Name())
				if err != nil {
					log.Error().Err(err).Str("pool", pool.Name()).Msg("failed to get devices for pool")
					continue
				}

				var deviceNames []string
				for _, device := range devices {
					deviceNames = append(deviceNames, device.Path())
				}

				usage, err := disk.UsageWithContext(ctx, pool.Path())
				if err != nil {
					log.Error().Err(err).Str("pool", pool.Name()).Msg("failed to get pool usage")
					continue
				}
				stats, err := disk.IOCountersWithContext(ctx, deviceNames...)
				if err != nil {
					log.Error().Err(err).Str("pool", pool.Name()).Msg("failed to get io stats for pool")
					continue
				}

				poolStats := pkg.PoolStats{
					UsageStat: *usage,
					Counters:  stats,
				}

				values[pool.Name()] = poolStats
			}

			select {
			case <-ctx.Done():
				return
			case ch <- values:
			}
		}

	}()

	return ch
}

func (s *Module) periodicallyCheckDiskShutdown() {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for {
			<-ticker.C
			log.Info().Msg("Checking pools for disks that should be shutdown...")
			s.shutdownDisks()
		}
	}()
}

// shutdownDisks will check the disks power status.
// If a disk is on and it is not mounted then it is not supposed to be on, turn it off
func (s *Module) shutdownDisks() {
	for _, pool := range s.ssds {
		device := pool.Device()
		log.Debug().Msgf("checking device: %s", device.Path())
		on, err := checkDiskPowerStatus(device.Path())
		if err != nil {
			log.Err(err).Msgf("error occurred while checking disk power status")
			continue
		}

		_, err = pool.Mounted()
		if err == nil || !on {
			continue
		}

		log.Debug().Msgf("shutting down device %s because it is not mounted and the device is on", device.Path())
		err = pool.Shutdown()
		if err != nil {
			log.Err(err).Msgf("failed to shutdown device %s", device.Path())
			continue
		}
	}
}

func checkDiskPowerStatus(path string) (bool, error) {
	output, err := exec.Command("smartctl", "-i", "-n", "standby", path).Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	blocks := strings.Split(string(output), "\n\n")
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		if strings.Contains(block, "ACTIVE") {
			return true, nil
		}
	}

	return false, nil
}
