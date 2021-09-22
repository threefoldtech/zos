package qsfsd

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	qsfsFlist       = "https://hub.grid.tf/azmy.3bot/qsfs.flist"
	qsfsContainerNS = "qsfs"
)

type QSFS struct {
	cl zbus.Client

	mountsPath string
}

func New(ctx context.Context, cl zbus.Client, root string) (pkg.QSFSD, error) {
	mountPath := filepath.Join(root, "mounts")
	err := os.MkdirAll(mountPath, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make qsfs mounts dir")
	}

	return &QSFS{
		cl:         cl,
		mountsPath: mountPath,
	}, nil
}

func (q *QSFS) Mount(wlID string, cfg zos.QuatumSafeFS) (string, error) {
	networkd := stubs.NewNetworkerStub(q.cl)
	flistd := stubs.NewFlisterStub(q.cl)
	contd := stubs.NewContainerModuleStub(q.cl)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	netns, err := networkd.QSFSPrepare(ctx, wlID)
	defer func() {
		if err != nil {
			err := networkd.QSFSDestroy(ctx, wlID)
			if err != nil {
				log.Error().Err(err).Msg("failed to cleanup qsfs after failure")
			}
		}
	}()
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare qsfs")
	}
	flistPath, err := flistd.Mount(ctx, wlID, qsfsFlist, pkg.MountOptions{
		ReadOnly: false,
		Limit:    cfg.Cache,
		Storage:  "zdb://hub.grid.tf:9900",
	})
	q.writeQSFSConfig(flistPath, cfg.Config)
	if err != nil {
		return "", errors.Wrap(err, "failed to mount qsfs flist")
	}
	mountPath := q.mountPath(wlID)
	fusePath, err := q.prepareMountPath(mountPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare mount path")
	}
	cont := pkg.Container{
		Name:        wlID, // what should the name be?
		RootFS:      flistPath,
		Entrypoint:  "/sbin/zinit init",
		Interactive: false,
		Network:     pkg.NetworkInfo{Namespace: netns},
		Mounts: []pkg.MountInfo{
			{
				Source: fusePath,
				Target: "/mnt",
				Shared: true,
			},
		},
		Elevated: true,
	}
	_, err = contd.Run(
		ctx,
		qsfsContainerNS,
		cont,
	)
	return fusePath, nil
}
func (q *QSFS) Unmount(wlID string) error {
	networkd := stubs.NewNetworkerStub(q.cl)
	flistd := stubs.NewFlisterStub(q.cl)
	contd := stubs.NewContainerModuleStub(q.cl)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	// listing all containers and matching the name looks like a lot of work
	if err := contd.Delete(ctx, qsfsContainerNS, pkg.ContainerID(wlID)); err != nil {
		log.Error().Err(err).Msg("failed to delete qsfs container")
	}
	mountPath := q.mountPath(wlID)
	fusePath := filepath.Join(mountPath, "mnt")
	if err := syscall.Unmount(fusePath, 0); err != nil {
		log.Error().Err(err).Msg("failed to unmount mount path")
	}
	if err := syscall.Unmount(mountPath, 0); err != nil {
		log.Error().Err(err).Msg("failed to unmount mount path")
	}
	if err := os.RemoveAll(mountPath); err != nil {
		log.Error().Err(err).Msg("failed to remove mountpath dir")
	}
	if err := flistd.Unmount(ctx, wlID); err != nil {
		log.Error().Err(err).Msg("failed to unmount flist")
	}

	if err := networkd.QSFSDestroy(ctx, wlID); err != nil {
		log.Error().Err(err).Msg("failed to unmount flist")
	}
	return nil
}

func (q *QSFS) mountPath(wlID string) string {
	return filepath.Join(q.mountsPath, wlID)
}

func (q *QSFS) prepareMountPath(path string) (string, error) {
	fusePath := filepath.Join(path, "mnt")
	if err := os.Mkdir(path, 0644); err != nil {
		return "", err
	}
	if err := os.Mkdir(fusePath, 0644); err != nil {
		return "", err
	}
	if err := syscall.Mount(path, path, "bind", syscall.MS_BIND, ""); err != nil {
		return "", err
	}
	if err := syscall.Mount("none", path, "", syscall.MS_SHARED, ""); err != nil {
		if err := syscall.Unmount(path, 0); err != nil {
			log.Error().Err(err).Msg("failed to unmount after failure")
		}
		return "", err
	}
	return fusePath, nil
}

func (q *QSFS) writeQSFSConfig(root string, cfg zos.QuantumSafeFSConfig) error {
	cfgPath := filepath.Join(root, "data/zstor.toml")
	f, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "couldn't open zstor config file")
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return errors.Wrap(err, "failed to convert config to yaml")
	}

	return nil
}
