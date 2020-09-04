package provision

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zdb"
	"golang.org/x/net/context"
)

const (
	containerdSock = "/run/containerd/containerd.sock"
)

// we declare this method as a variable so we can
// mock it in testing.
func initZdbConnection(id string) zdb.Client {
	socket := fmt.Sprintf("unix://%s/zdb.sock", socketDir(id))
	return zdb.New(socket)
}

func socketDir(containerID string) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
}

// CleanupResources cleans up unused resources
func cleanupResources(msgBrokerCon string) error {
	toSave, toDelete, err := checkContainers(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "failed to check containers")
	}

	ctx := context.TODO()
	utils := filesystem.NewUtils()
	pools, err := ioutil.ReadDir("/mnt")
	if err != nil {
		return errors.Wrap(err, "failed to list available pools")
	}

	for _, pool := range pools {
		if !pool.IsDir() {
			continue
		}
		path := filepath.Join("/mnt", pool.Name())
		subvols, err := utils.SubvolumeList(ctx, path)
		if err != nil {
			// A Pool might not be mounted here, skip it
			log.Error().Err(err).Msgf("failed to list available subvolumes in: %s", path)
			continue
		}

		qgroups, err := utils.QGroupList(ctx, path)
		if err != nil {
			return errors.Wrapf(err, "failed to list available cgroups in: %s", path)
		}

		for _, subvol := range subvols {

			qgroup, ok := qgroups[fmt.Sprintf("0/%d", subvol.ID)]
			if !ok {
				log.Info().Msgf("skipping volume '%s' has no assigned quota", subvol.Path)
				continue
			}

			// now, is this subvol in one of the toDelete ?
			if _, ok := toDelete[filepath.Base(subvol.Path)]; ok {
				// delete the subvolume
				log.Info().Msgf("deleting subvolume %s", subvol.Path)
				if err := utils.SubvolumeRemove(ctx, filepath.Join(path, subvol.Path)); err != nil {
					log.Err(err).Msgf("failed to delete subvol '%s'", subvol.Path)
				}
				if err := utils.QGroupDestroy(ctx, qgroup.ID, path); err != nil {
					log.Err(err).Msgf("failed to delete qgroup: '%s'", qgroup.ID)
				}
				continue
			}

			// now, is this subvol in one of the toSave ?
			if _, ok := toSave[filepath.Base(subvol.Path)]; ok {
				log.Info().Msgf("skipping volume '%s' is used", subvol.Path)
				continue
			}

			// Don't delete zos-cache!
			if subvol.Path == "zos-cache" {
				continue
			}

			// delete the subvolume
			log.Info().Msgf("deleting subvolume %s", subvol.Path)
			if err := utils.SubvolumeRemove(ctx, filepath.Join(path, subvol.Path)); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", subvol.Path)
			}
			if err := utils.QGroupDestroy(ctx, qgroup.ID, path); err != nil {
				log.Err(err).Msgf("failed to delete qgroup: '%s'", qgroup.ID)
			}
		}
	}

	return nil
}

// checks running containers for subvolumes that might need to be saved because they are used
// and subvolumes that might need to be deleted because they have no attached container anymore
func checkContainers(msgBrokerCon string) (map[string]struct{}, map[string]struct{}, error) {
	toSave := make(map[string]struct{})
	toDelete := make(map[string]struct{})

	client, err := containerd.New(containerdSock)
	if err != nil {
		log.Err(err).Msgf("failed to create containerd connection")
		return nil, nil, err
	}

	ns, err := client.NamespaceService().List(context.Background())
	if err != nil {
		log.Err(err).Msgf("failed to list namespaces")
		return nil, nil, err
	}

	for _, ns := range ns {
		log.Info().Msgf("Checking namespace %s", ns)
		ctx := namespaces.WithNamespace(context.Background(), ns)
		crts, err := client.Containers(ctx, "")
		if err != nil {
			log.Err(err).Msgf("failed to list containers")
			return nil, nil, err
		}

		var zdbNamespaces []string

		for _, ctr := range crts {
			log.Info().Msgf("container ID %s", ctr.ID())
			if ns == "zdb" {
				zdbCl := initZdbConnection(ctr.ID())
				defer zdbCl.Close()
				if err := zdbCl.Connect(); err != nil {
					log.Err(err).Msgf("failed to connect to 0-db: %s", ctr.ID())
					continue
				}

				zdbNamespaces, err = zdbCl.Namespaces()
				if err != nil {
					log.Err(err).Msg("failed to retrieve zdb namespaces")
					continue
				}
			}

			spec, _ := ctr.Spec(ctx)
			if err != nil {
				log.Err(err).Msg("failed to container spec")
				continue
			}

			toSave[filepath.Base(spec.Root.Path)] = struct{}{}
			for _, mnt := range spec.Mounts {
				if !strings.HasPrefix(mnt.Source, "/mnt/") {
					continue
				}

				if len(ns) == 1 && zdbNamespaces[0] == "default" {
					err := deleteZdbContainer(msgBrokerCon, pkg.ContainerID(ctr.ID()))
					if err != nil {
						log.Err(err).Msgf("failed to delete zdb container %s", ctr.ID())
						continue
					}
					toDelete[filepath.Base(mnt.Source)] = struct{}{}
				} else {
					toSave[filepath.Base(mnt.Source)] = struct{}{}
				}
			}
		}
	}

	return toSave, toDelete, nil
}

func deleteZdbContainer(msgBrokerCon string, containerID pkg.ContainerID) error {
	zbusCl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to connect to message broker server")
	}

	container := stubs.NewContainerModuleStub(zbusCl)
	flist := stubs.NewFlisterStub(zbusCl)

	info, err := container.Inspect("zdb", containerID)
	if err != nil {
		log.Error().Err(err).Str("container", string(containerID)).Msg("failed to inspect container for decomission")
		return err
	}

	if err := container.Delete("zdb", containerID); err != nil {
		return errors.Wrapf(err, "failed to delete container %s", containerID)
	}

	rootFS := info.RootFS
	if info.Interactive {
		rootFS, err = findRootFS(info.Mounts)
		if err != nil {
			return err
		}
	}

	if err := flist.Umount(rootFS); err != nil {
		return errors.Wrapf(err, "failed to unmount flist at %s", rootFS)
	}

	return nil
}

func findRootFS(mounts []pkg.MountInfo) (string, error) {
	for _, m := range mounts {
		if m.Target == "/sandbox" {
			return m.Source, nil
		}
	}

	return "", fmt.Errorf("rootfs flist mountpoint not found")
}
