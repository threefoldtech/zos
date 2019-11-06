package filesystem

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	zosV1PoolPrefix = "sp_"
)

var (
	zeros [512]byte
)

func wipe(ctx context.Context, device *Device, exe executer) error {
	//we first make an msdos parition, because we know it's 512 bytes
	//creating the msdos partition table also makes us validate that this
	//is indeed a valid block device
	if _, err := exe.run(ctx, "parted", "-s", device.Path, "mktable", "msdos"); err != nil {
		return errors.Wrap(err, "failed to create msdos partition table")
	}

	//then we write down this exact amount to device
	return ioutil.WriteFile(device.Path, zeros[:], 0660)
}

// Migrate is a simple migration routine that makes sure we have
// fresh disks to use from zos v1, in case you moving
// away from older version
// After migration you should alway use the returned Device manager, or create
// a new one.
func Migrate(ctx context.Context, m DeviceManager) (DeviceManager, error) {
	return migrate(ctx, m, executerFunc(run))
}

func migrate(ctx context.Context, m DeviceManager, exe executer) (DeviceManager, error) {
	devices, err := m.Raw(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "migration failed, couldn't list disks")
	}

	var destroyMe []Device
loop:
	for _, device := range devices {
		if strings.HasPrefix(device.Label, zosV1PoolPrefix) {
			destroyMe = append(destroyMe, device)
			continue
		}

		for _, child := range device.Children {
			if strings.HasPrefix(child.Label, zosV1PoolPrefix) {
				destroyMe = append(destroyMe, device)
				continue loop
			}
		}
	}

	for _, device := range destroyMe {
		log.Warn().Str("device", device.Path).Msg("wiping device")
		if err := wipe(ctx, &device, exe); err != nil {
			log.Error().Err(err).Str("device", device.Path).Msg("failed to wipe disk")
		}
	}
	if len(destroyMe) > 0 {
		if out, err := exe.run(ctx, "sync"); err != nil {
			log.Error().Err(err).Str("out", string(out)).Msg("failed sync devices")
		}
		if out, err := exe.run(ctx, "partprobe"); err != nil {
			log.Error().Err(err).Str("out", string(out)).Msg("failed to partprobe")
		}

	}

	return m.Reset(), nil
}
