package storage

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
	"github.com/threefoldtech/zos/pkg/storage/zdbpool"
)

func isZdbVolume(volume filesystem.Volume) bool {
	return strings.HasPrefix(volume.Name(), zdbPoolPrefix)
}

func (s *storageModule) Find(nsID string) (allocation pkg.Allocation, err error) {
	for _, pool := range s.volumes {
		volumes, err := pool.Volumes()
		if err != nil {
			return allocation, errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
		}

		for _, volume := range volumes {
			// skip all non-zdb volume
			if !isZdbVolume(volume) {
				continue
			}

			zdb := zdbpool.New(volume.Path())

			if !zdb.Exists(nsID) {
				continue
			}

			// we found the namespace
			allocation = pkg.Allocation{
				VolumeID:   volume.Name(),
				VolumePath: volume.Path(),
			}

			return allocation, nil
		}
	}

	return pkg.Allocation{}, fmt.Errorf("not found")
}

// Allocate is responsible to make sure the subvolume used by a 0-db as enough storage capacity
// of specified size, type and mode
// it returns the volume ID and its path or an error if it couldn't allocate enough storage
func (s *storageModule) Allocate(nsID string, diskType pkg.DeviceType, size uint64, mode pkg.ZDBMode) (allocation pkg.Allocation, err error) {
	log := log.With().
		Str("type", string(diskType)).
		Uint64("size", size).
		Logger()

	log.Info().Msg("try to allocation space for 0-DB")

	for _, pool := range s.volumes {

		// skip pool with wrong disk type
		if pool.Type() != diskType {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return allocation, errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
		}

		for _, volume := range volumes {
			// skip all non-zdb volume
			if !isZdbVolume(volume) {
				continue
			}

			zdb := zdbpool.New(volume.Path())

			if !zdb.Exists(nsID) {
				continue
			}

			// we found the namespace
			allocation = pkg.Allocation{
				VolumeID:   volume.Name(),
				VolumePath: volume.Path(),
			}

			return allocation, nil
		}
	}

	type Candidate struct {
		filesystem.Volume
		Free uint64
	}

	var candidates []Candidate
	// okay, so this is a new allocation
	for _, pool := range s.volumes {
		// skip pool with wrong disk type
		if pool.Type() != diskType {
			continue
		}

		usage, err := pool.Usage()
		if err != nil {
			return allocation, errors.Wrapf(err, "failed to read usage of pool %s", pool.Name())
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return allocation, errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
		}

		for _, volume := range volumes {
			// skip all non-zdb volume
			if !isZdbVolume(volume) {
				continue
			}

			zdb := zdbpool.New(volume.Path())

			reserved, err := zdb.Reserved()
			if err != nil {
				return allocation, errors.Wrapf(err, "failed to list namespaces from volume '%s'", volume.Path())
			}

			if reserved+size > usage.Size {
				// not enough space on this namespace
				continue
			}

			candidates = append(
				candidates,
				Candidate{
					Volume: volume,
					Free:   usage.Size - (reserved + size),
				})
		}
	}

	var volume filesystem.Volume
	if len(candidates) > 0 {
		// reverse sort by free space
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Free > candidates[j].Free
		})

		volume = candidates[0]
	} else {
		// no candidates, so we have to try to create a new subvolume.
		name, err := genZDBPoolName()
		if err != nil {
			return allocation, errors.Wrap(err, "failed to generate new sub-volume name")
		}

		volume, err = s.createSubvol(size, name, diskType)
		if err != nil {
			return allocation, errors.Wrap(err, "failed to create sub-volume")
		}
	}

	zdb := zdbpool.New(volume.Path())

	if err := zdb.Create(nsID, "", size); err != nil {
		return allocation, errors.Wrapf(err, "failed to create namespace directory: '%s/%s'", volume.Path(), nsID)
	}

	return pkg.Allocation{
		VolumeID:   volume.Name(),
		VolumePath: volume.Path(),
	}, nil

}

const zdbPoolPrefix = "zdb"

func genZDBPoolName() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	name := zdbPoolPrefix + id.String()
	return name, nil
}
