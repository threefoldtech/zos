package storage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/pkg/errors"

	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/storage/filesystem"
)

// Allocate is responsible to make sure the subvolume used by a 0-db as enough storage capacity
// of specified size, type and mode
// it returns the volume ID and its path or an error if it couldn't allocate enough storage
func (s *storageModule) Allocate(diskType modules.DeviceType, size uint64, mode modules.ZDBMode) (string, string, error) {
	// try to find an existing zdb volume that has still enough storage available
	// if we find it, grow the quota by the requested size
	// if we don't, pick a new pool and create a zdb volume on it with the requested size

	var (
		v   filesystem.Volume
		err error
	)

	for _, pool := range s.volumes {

		// skip pool with wrong disk type
		if pool.Type() != diskType {
			continue
		}

		volumes, err := pool.Volumes()
		if err != nil {
			return "", "", errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
		}

		for _, volume := range volumes {

			// skip all non-zdb volume
			if !strings.HasPrefix(volume.Name(), "_zdb") {
				continue
			}

			usage, err := volume.Usage()
			if err != nil {
				return "", "", errors.Wrapf(err, "failed to read usage of volume %s", volume.Name())
			}

			// skip pool with not enough space
			if usage.Used+size > usage.Size {
				continue
			}

			// found a valid volume !
			v = volume
			break
		}

		if v != nil {
			break
		}
	}

	if v == nil {
		name, err := genZDBPoolName()
		if err != nil {
			return "", "", errors.Wrap(err, "failed to generate new sub-volume name")
		}

		v, err = s.createSubvol(size, name, diskType)
		if err != nil {
			return "", "", errors.Wrap(err, "failed to create sub-volume")
		}
	}

	usage, err := v.Usage()
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to read usage of volume %s", v.Name())
	}

	// grow its limit
	if err := v.Limit(usage.Size + size); err != nil {
		return "", "", errors.Wrapf(err, "failed to grow limit of volume %s", v.Name())
	}

	return v.Name(), v.Path(), nil
}

// Claim let the system claim the allocated storage used by a 0-db namespace
func (s *storageModule) Claim(name string, size uint64) error {
	var (
		v   filesystem.Volume
		err error
	)

	for _, pool := range s.volumes {

		volumes, err := pool.Volumes()
		if err != nil {
			return errors.Wrapf(err, "failed to list volume on pool %s", pool.Name())
		}

		for _, volume := range volumes {
			if volume.Name() == name {
				v = volume
				break
			}
		}

		if v != nil {
			break
		}
	}

	if v == nil {
		return fmt.Errorf("volume named %s not found", name)
	}

	usage, err := v.Usage()
	if err != nil {
		return errors.Wrapf(err, "failed to read usage of volume %s", v.Name())
	}

	// shrink the limit
	if err := v.Limit(usage.Size - size); err != nil {
		return errors.Wrapf(err, "failed to grow limit of volume %s", v.Name())
	}

	return nil
}

func genZDBPoolName() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	name := "_zsb" + id.String()
	return name, nil
}
