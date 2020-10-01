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
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/provision/common"
	"github.com/threefoldtech/zos/pkg/storage"
	"github.com/threefoldtech/zos/pkg/storage/filesystem"
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
func CleanupResources(msgBrokerCon string) error {
	client, err := app.ExplorerClient()
	if err != nil {
		return err
	}

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
			log.Info().Msgf("checking subvol %s", subvol.Path)
			// Don't delete zos-cache!
			if subvol.Path == storage.CacheLabel {
				continue
			}

			qgroup, ok := qgroups[fmt.Sprintf("0/%d", subvol.ID)]
			if !ok {
				log.Info().Msgf("skipping volume '%s' has no assigned quota", subvol.Path)
				continue
			}

			// Now, is this subvol in one of the toSave ?
			if _, ok := toSave[filepath.Base(subvol.Path)]; ok {
				log.Info().Msgf("skipping volume '%s' is used", subvol.Path)
				continue
			}

			// Now, is this subvol in one of the toDelete ?
			if _, ok := toDelete[filepath.Base(subvol.Path)]; ok {
				// delete the subvolume
				delete := checkReservationToDelete(subvol.Path, client)
				if delete {
					log.Info().Msgf("deleting subvolume %s", subvol.Path)
					if err := utils.SubvolumeRemove(ctx, filepath.Join(path, subvol.Path)); err != nil {
						log.Err(err).Msgf("failed to delete subvol '%s'", subvol.Path)
					}
					if err := utils.QGroupDestroy(ctx, qgroup.ID, path); err != nil {
						log.Err(err).Msgf("failed to delete qgroup: '%s'", qgroup.ID)
					}
					continue
				}
				log.Info().Msgf("skipping subvolume %s", subvol.Path)
				continue
			}

			// Is this subvol not in toDelete and not in toSave?
			// Check the explorer if it needs to be deleted
			delete := checkReservationToDelete(subvol.Path, client)
			if delete {
				log.Info().Msgf("deleting subvolume %s", subvol.Path)
				if err := utils.SubvolumeRemove(ctx, filepath.Join(path, subvol.Path)); err != nil {
					log.Err(err).Msgf("failed to delete subvol '%s'", subvol.Path)
				}
				if err := utils.QGroupDestroy(ctx, qgroup.ID, path); err != nil {
					log.Err(err).Msgf("failed to delete qgroup: '%s'", qgroup.ID)
				}
				continue
			}
			log.Info().Msgf("skipping subvolume %s", subvol.Path)
		}
	}

	return nil
}

func checkReservationToDelete(path string, cl *client.Client) bool {
	log.Info().Msgf("checking explorer for reservation: %s", path)
	reservation, err := cl.Workloads.NodeWorkloadGet(path)
	if err != nil {
		var hErr client.HTTPError
		if ok := errors.As(err, &hErr); ok {
			resp := hErr.Response()
			// If reservation is not found it should be deleted
			if resp.StatusCode == 404 {
				return true
			}
		}
		return false
	}

	if reservation.GetNextAction() == workloads.NextActionDelete {
		log.Info().Msgf("subvolume with path %s has next action to delete", path)
		return true
	}

	return false
}

// checks running containers for subvolumes that might need to be saved because they are used
// and subvolumes that might need to be deleted because they have no attached container anymore
func checkContainers(msgBrokerCon string) (map[string]struct{}, map[string]struct{}, error) {
	toSave := make(map[string]struct{})
	toDelete := make(map[string]struct{})
	zbus, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return nil, nil, err
	}

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

			spec, err := ctr.Spec(ctx)
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
					err := common.DeleteZdbContainer(pkg.ContainerID(ctr.ID()), zbus)
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
