package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zdb"
	"golang.org/x/net/context"
)

func execute(cmd string, args ...string) ([]byte, error) {
	exe := exec.Command(cmd, args...)
	output, err := exe.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "execute '%v %+v': %v", cmd, args, string(output))
	}

	return output, nil
}

func lines(input []byte) []string {
	input = bytes.TrimSpace(input)
	if len(input) == 0 {
		return nil
	}

	return strings.Split(string(input), "\n")
}

func must(input []byte, err error) []byte {
	if err != nil {
		panic(err)
	}

	return input
}

type container struct {
	ID   string `json:"ID"`
	Spec struct {
		Root struct {
			Path string `json:"path"`
		} `json:"root"`
		Mounts []struct {
			Source string `json:"source"`
		} `json:"mounts"`
	} `json:"Spec"`
}

// we declare this method as a variable so we can
// mock it in testing.
var zdbConnection = func(id string) zdb.Client {
	socket := fmt.Sprintf("unix://%s/zdb.sock", socketDir(id))
	return zdb.New(socket)
}

func socketDir(containerID string) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
}

// CleanupResources cleans up unused resources
func CleanupResources() error {
	toSave := make(map[string]struct{})
	toDelete := make(map[string]struct{})

	for _, ns := range lines(must(execute("ctr", "namespace", "ls", "-q"))) {
		log.Info().Msgf("Checking namespace %s", ns)
		for _, cnt := range lines(must(execute("ctr", "-n", ns, "container", "ls", "-q"))) {
			bytes := must(execute("ctr", "-n", ns, "container", "info", cnt))
			var container container
			if err := json.Unmarshal(bytes, &container); err != nil {
				panic(err)
			}

			zdbCl := zdbConnection(container.ID)
			defer zdbCl.Close()
			if err := zdbCl.Connect(); err != nil {
				return errors.Wrapf(err, "failed to connect to 0-db: %s", container.ID)
			}

			ns, err := zdbCl.Namespaces()
			if err != nil {
				return errors.Wrap(err, "failed to retrieve zdb namespaces")
			}

			toSave[filepath.Base(container.Spec.Root.Path)] = struct{}{}
			for _, mnt := range container.Spec.Mounts {
				if strings.HasPrefix(mnt.Source, "/mnt/") {
					log.Info().Msgf("ZDB namespaces length: %d in path %s", len(ns), mnt.Source)
					if len(ns) == 1 && ns[0] == "default" {
						err := deleteZdbContainer(pkg.ContainerID(container.ID))
						if err != nil {
							return errors.Wrapf(err, "failed to delete zdb container %s", container.ID)
						}
						toDelete[filepath.Base(mnt.Source)] = struct{}{}
					} else {
						toSave[filepath.Base(mnt.Source)] = struct{}{}
					}
				}
			}
		}
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

			// we only handle volumes that are 256MiB or 10MiB
			if qgroup.MaxRfer != 268435456 && qgroup.MaxRfer != 10485760 {
				// if the subvolume is a zdb and has 0 maxRfer, don't skip here. It might need to be deleted
				if !strings.HasPrefix(subvol.Path, "zdb") && qgroup.MaxRfer != 0 {
					log.Info().Msgf("skipping volume '%s' is of size: %d", subvol.Path, qgroup.MaxRfer)
					continue
				}
			}

			// now, is this subvol in one of the toSave ?
			if _, ok := toSave[filepath.Base(subvol.Path)]; ok {
				log.Info().Msgf("skipping volume '%s' is used", subvol.Path)
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

func deleteZdbContainer(containerID pkg.ContainerID) error {
	zbusCl, err := zbus.NewRedisClient("unix:///var/run/redis.sock")
	if err != nil {
		log.Fatal().Err(err).Msg("fail to connect to message broker server")
	}

	container := stubs.NewContainerModuleStub(zbusCl)
	flist := stubs.NewFlisterStub(zbusCl)

	info, err := container.Inspect("zdb", containerID)
	if err == nil {
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

	} else {
		log.Error().Err(err).Str("container", string(containerID)).Msg("failed to inspect container for decomission")
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
