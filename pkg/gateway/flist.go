package gateway

import (
	"context"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	flist = "https://hub.grid.tf/omar0.3bot/traefik.flist"
)

// ensureTraefikBin makes sure traefik flist is mounted.
// TODO: we need to "update" traefik and restart the service
// if new version is available!
func ensureTraefikBin(ctx context.Context, cl zbus.Client) (string, bool, error) {
	const bin = "traefik"
	flistd := stubs.NewFlisterStub(cl)

	mnt, err := flistd.Mount(ctx, bin, flist, pkg.ReadOnlyMountOptions)
	if err != nil {
		return "", false, errors.Wrap(err, "failed to mount traefik flist")
	}
	oldHash, err := flistd.HashFromRootPath(ctx, mnt)
	if err != nil {
		return "", false, errors.Wrap(err, "failed to get old traefik flist hash")
	}
	hash, err := flistd.FlistHash(ctx, flist)
	if err != nil {
		return "", false, errors.Wrap(err, "failed to get traefik flist hash")
	}
	log.Debug().Str("old", oldHash).Str("new", hash).Msg("flist changes")
	updated := false
	if oldHash != hash {
		updated = true
		if err := flistd.Unmount(ctx, bin); err != nil {
			return "", false, errors.Wrap(err, "failed to unmount old traefik flist")
		}
		mnt, err = flistd.Mount(ctx, bin, flist, pkg.ReadOnlyMountOptions)
		if err != nil {
			return "", false, errors.Wrap(err, "failed to mount new traefik flist")
		}
	}
	return filepath.Join(mnt, bin), updated, nil
}
