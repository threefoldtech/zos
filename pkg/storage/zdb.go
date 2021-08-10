package storage

import (
	"fmt"

	"github.com/threefoldtech/zos/pkg"
)

//Devices list all "allocated" devices
func (m *Module) Devices() ([]pkg.Device, error) {
	return nil, fmt.Errorf("not implemented")
}

// DeviceAllocate allocates a new device (formats and give a new ID)
func (m *Module) DeviceAllocate() (pkg.Device, error) {
	return pkg.Device{}, fmt.Errorf("not implemented")
}

// import (
// 	"fmt"
// 	"sort"

// 	"github.com/rs/zerolog/log"

// 	"github.com/google/uuid"

// 	"github.com/pkg/errors"

// 	"github.com/threefoldtech/zos/pkg"
// 	"github.com/threefoldtech/zos/pkg/gridtypes"
// 	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
// 	"github.com/threefoldtech/zos/pkg/storage/filesystem"
// 	"github.com/threefoldtech/zos/pkg/storage/zdbpool"
// )

// // Find finds a zdb namespace allocation
// func (s *Module) Find(nsID string) (allocation pkg.Allocation, err error) {
// 	for _, pool := range s.ssds {
// 		if _, err := pool.Mounted(); err != nil {
// 			continue
// 		}

// 		volumes, err := pool.Volumes()
// 		if err != nil {
// 			return allocation, errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
// 		}

// 		for _, volume := range volumes {
// 			// skip all non-zdb volume
// 			if !filesystem.IsZDBVolume(volume) {
// 				continue
// 			}

// 			zdb := zdbpool.New(volume.Path())

// 			if !zdb.Exists(nsID) {
// 				continue
// 			}

// 			// we found the namespace
// 			allocation = pkg.Allocation{
// 				VolumeID:   volume.Name(),
// 				VolumePath: volume.Path(),
// 			}

// 			return allocation, nil
// 		}
// 	}

// 	return pkg.Allocation{}, fmt.Errorf("not found")
// }

// // Allocate is responsible to make sure the subvolume used by a 0-db as enough storage capacity
// // of specified size, type and mode
// // it returns the volume ID and its path or an error if it couldn't allocate enough storage
// func (s *Module) Allocate(nsID string, diskType pkg.DeviceType, size gridtypes.Unit, mode pkg.ZDBMode) (allocation pkg.Allocation, err error) {
// 	log := log.With().
// 		Str("type", string(diskType)).
// 		Uint64("size", uint64(size)).
// 		Str("mode", string(mode)).
// 		Logger()

// 	if diskType != zos.HDDDevice && diskType != zos.SSDDevice {
// 		return allocation, pkg.ErrInvalidDeviceType{DeviceType: diskType}
// 	}

// 	log.Info().Msg("try to allocation space for 0-DB")

// 	// Initially check if the namespace already exists
// 	// if so, return the allocation
// 	for _, pool := range s.hdds {
// 		if _, err := pool.Mounted(); err != nil {
// 			continue
// 		}

// 		// skip pool with wrong disk type
// 		if pool.Type() != diskType {
// 			continue
// 		}

// 		volumes, err := pool.Volumes()
// 		if err != nil {
// 			return allocation, errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
// 		}

// 		for _, volume := range volumes {

// 			// skip all non-zdb volume
// 			if !filesystem.IsZDBVolume(volume) {
// 				continue
// 			}

// 			zdb := zdbpool.New(volume.Path())

// 			if !zdb.Exists(nsID) {
// 				continue
// 			}

// 			// we found the namespace
// 			allocation = pkg.Allocation{
// 				VolumeID:   volume.Name(),
// 				VolumePath: volume.Path(),
// 			}

// 			return allocation, nil
// 		}
// 	}

// 	targetMode := zdbpool.IndexModeKeyValue
// 	if mode == zos.ZDBModeSeq {
// 		targetMode = zdbpool.IndexModeSequential
// 	}

// 	// check for candidates in mounted pools first
// 	candidates, err := s.checkForZDBCandidateVolumes(size, diskType, targetMode)
// 	if err != nil {
// 		log.Error().Err(err).Msgf("failed to search volumes on mounted pools")
// 		return allocation, err
// 	}

// 	log.Debug().Msgf("Found %d candidate volumes in mounted pools", len(candidates))

// 	var volume filesystem.Volume
// 	if len(candidates) > 0 {
// 		// reverse sort by free space
// 		sort.Slice(candidates, func(i, j int) bool {
// 			return candidates[i].Free > candidates[j].Free
// 		})

// 		volume = candidates[0]
// 	} else {
// 		// no candidates, so we have to try to create a new subvolume.
// 		// and start a new zdb instance
// 		name, err := genZDBPoolName()
// 		if err != nil {
// 			return allocation, errors.Wrap(err, "failed to generate new sub-volume name")
// 		}

// 		// we create the zdb volume without configuring a quota
// 		// the used size will the computed from the 0-db namespaces themselves
// 		volume, err = s.createSubvol(size, name)
// 		if err != nil {
// 			return allocation, errors.Wrap(err, "failed to create sub-volume")
// 		}
// 	}

// 	zdb := zdbpool.New(volume.Path())

// 	if err := zdb.Create(nsID, "", size); err != nil {
// 		return allocation, errors.Wrapf(err, "failed to create namespace directory: '%s/%s'", volume.Path(), nsID)
// 	}

// 	return pkg.Allocation{
// 		VolumeID:   volume.Name(),
// 		VolumePath: volume.Path(),
// 	}, nil

// }

// type zdbcandidate struct {
// 	filesystem.Volume
// 	Free uint64
// }

// func (s *Module) checkForZDBCandidateVolumes(size gridtypes.Unit, poolType pkg.DeviceType, targetMode zdbpool.IndexMode) ([]zdbcandidate, error) {
// 	var candidates []zdbcandidate
// 	for _, pool := range s.ssds {
// 		// ignore pools that are not mounted for now
// 		if _, err := pool.Mounted(); err != nil {
// 			continue
// 		}

// 		// ignore pools which don't have the right device type
// 		if pool.Type() != poolType {
// 			continue
// 		}
// 		log.Debug().Msgf("checking pool %s for space", pool.Name())

// 		usage, err := pool.Usage()
// 		if err != nil {
// 			log.Error().Err(err).Msgf("failed to read usage of pool %s", pool.Name())
// 			return nil, err
// 		}

// 		volumes, err := pool.Volumes()
// 		if err != nil {
// 			log.Error().Err(err).Msgf("failed to list volume on pool %s", pool.Name())
// 			return nil, err
// 		}

// 		for _, volume := range volumes {
// 			// skip all non-zdb volume
// 			if !filesystem.IsZDBVolume(volume) {
// 				continue
// 			}

// 			volumeUsage, err := volume.Usage()
// 			if err != nil {
// 				log.Error().Err(err).Msgf("failed to list namespaces from volume '%s'", volume.Path())
// 				return nil, err
// 			}

// 			if volumeUsage.Size+uint64(size) > usage.Size {
// 				// not enough space on this volume
// 				continue
// 			}

// 			zdb := zdbpool.New(volume.Path())

// 			// check if the mode is the same
// 			indexMode, err := zdb.IndexMode("default")
// 			if err != nil {
// 				log.Err(err).Str("namespace", "default").Msg("failed to read index mode")
// 				continue
// 			}

// 			if indexMode != targetMode {
// 				log.Info().Msg("skip because wrong mode")
// 				continue
// 			}

// 			candidates = append(
// 				candidates,
// 				zdbcandidate{
// 					Volume: volume,
// 					Free:   usage.Size - (volumeUsage.Size + uint64(size)),
// 				})
// 		}
// 	}
// 	return candidates, nil
// }

// // ZDBPoolPrefix is the prefix used for the subvolume create for 0-DB containers
// const ZDBPoolPrefix = "zdb"

// func genZDBPoolName() (string, error) {
// 	id, err := uuid.NewRandom()
// 	if err != nil {
// 		return "", err
// 	}
// 	name := ZDBPoolPrefix + id.String()
// 	return name, nil
// }
