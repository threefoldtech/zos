package provision

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision/common"
	"github.com/threefoldtech/zos/pkg/storage"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zdb"
	"golang.org/x/net/context"
)

type Janitor struct {
	zbus zbus.Client

	cache    ReservationCache
	explorer *client.Client
}

// CleanupResources cleans up unused resources
func (j *Janitor) CleanupResources(ctx context.Context) error {

	storaged := stubs.NewStorageModuleStub(j.zbus)

	toSave, err := j.checkContainers(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check containers")
	}

	// ListFilesystems do not return the special cache and vdisk filesystem
	// so we are safe to process everything that is returned
	fss, err := storaged.ListFilesystems()
	if err != nil {
		return err
	}

	for _, fs := range fss {
		log.Info().Msgf("checking subvol %s", fs.Path)

		// Now, is this subvol in one of the toSave ?
		if _, ok := toSave[fs.Path]; ok {
			log.Info().Msgf("skipping volume '%s' is used", fs.Path)
			continue
		}

		if len(fs.Name) == 64 {
			// if the fs is not used by any container and its name is 64 character long
			// they are left over of old containers when flistd used to generate random names
			// for the container root flist subvolumes
			log.Info().Msgf("delete root container flist subvolume '%s'", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}

		if strings.HasPrefix(fs.Name, storage.ZDBPoolPrefix) {
			// we can safely delete this one because it is not used by any container
			// this is ensured line 46
			log.Info().Msgf("delete left over 0-DB subvolume '%s'", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}

		if fs.Name == "fcvms" {
			// left over from testing during vm module development
			log.Info().Msgf("delete fcvm subvolume '%s'", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}

		// Is this subvol not in toSave?
		// Check the explorer if it needs to be deleted
		delete, err := j.checkToDeleteExplorer(fs.Name)
		if err != nil {
			//TODO: handle error here
		}
		if delete {
			log.Info().Msgf("deleting subvolume %s", fs.Path)
			if err := storaged.ReleaseFilesystem(fs.Name); err != nil {
				log.Err(err).Msgf("failed to delete subvol '%s'", fs.Path)
			}
			continue
		}
		log.Info().Msgf("skipping subvolume %s", fs.Path)
	}

	return nil
}

func (j *Janitor) checkToDelete(id string) (bool, error) {
	reservation, err := j.cache.Get(id)
	if err != nil {
		// reservation not found in cache, so still a chance it's available on
		// the explorer. so we make this one call
		return j.checkToDeleteExplorer(id)
	}

	return reservation.Expired() || reservation.ToDelete, nil
}

func (j *Janitor) checkToDeleteExplorer(id string) (bool, error) {

	log.Info().Msgf("checking explorer for reservation: %s", id)
	reservation, err := j.explorer.Workloads.NodeWorkloadGet(id)
	if err != nil {
		var hErr client.HTTPError
		if ok := errors.As(err, &hErr); ok {
			resp := hErr.Response()
			// If reservation is not found it should be deleted
			if resp.StatusCode == 404 {
				return true, nil
			}
		}
		return false, err
	}

	nextAction := reservation.GetNextAction()
	if nextAction == workloads.NextActionDelete || nextAction == workloads.NextActionDeleted || nextAction == workloads.NextActionInvalid {
		log.Info().Msgf("workload %s has next action to delete / deleted or invalid", id)
		return true, nil
	}

	return false, nil
}

func (j *Janitor) checkZdbContainer(ctx context.Context, id string) error {
	con, err := newZdbConnection(id)
	if err != nil {
		return err
	}

	defer con.Close()
	namespaces, err := con.Namespaces()
	if err != nil {
		// we need to skip this zdb container for now we are not sure
		// if it has any used values.
		return errors.Wrap(err, "failed to list zdb namespace")
	}

	mapped := make(map[string]struct{})
	for _, namespace := range namespaces {
		if namespace == "default" {
			continue
		}

		mapped[namespace] = struct{}{}

		toDelete, err := j.checkToDelete(namespace)
		if err != nil {
			log.Error().Err(err).Str("zdb-namespace", namespace).Msg("failed to check if we should keep namespace")
			continue
		}

		if !toDelete {
			continue
		}

		if err := con.DeleteNamespace(namespace); err != nil {
			log.Error().Err(err).Str("zdb-namespace", namespace).Msg("failed to delete lingering zdb namespace")
		}

		delete(mapped, namespace)
	}

	if len(mapped) > 0 {
		// not all namespaces are deleted so we need to keep this
		// container instance
		return nil
	}

	// no more namespace to keep, so container can also go
	return common.DeleteZdbContainer(pkg.ContainerID(id), j.zbus)
}

func (j *Janitor) checkZdbContainers(ctx context.Context) error {
	containerd := stubs.NewContainerModuleStub(j.zbus)

	containers, err := containerd.List("zdb")
	if err != nil {
		return errors.Wrap(err, "failed to list zdb containers")
	}

	for _, containerID := range containers {
		if err := j.checkZdbContainer(ctx, string(containerID)); err != nil {
			log.Error().Err(err).Msg("failed to cleanup zdb container")
		}
	}

	return nil
}

// checks running containers for subvolumes that might need to be saved because they are used
// and subvolumes that might need to be deleted because they have no attached container anymore
func (j *Janitor) checkContainers(ctx context.Context) (map[string]struct{}, error) {
	toSave := make(map[string]struct{})

	contd := stubs.NewContainerModuleStub(j.zbus)

	cNamespaces, err := contd.ListNS()
	if err != nil {
		log.Err(err).Msgf("failed to list namespaces")
		return nil, err
	}

	for _, ns := range cNamespaces {

		containerIDs, err := contd.List(ns)
		if err != nil {
			log.Error().Err(err).Msg("failed to list container IDs")
			return nil, err
		}

		for _, id := range containerIDs {
			ctr, err := contd.Inspect(ns, id)
			if err != nil {
				log.Error().Err(err).Msgf("failed to inspect container %s", id)
				return nil, err
			}

			log.Info().Msgf("container ID %s", id)
			var zdbNamespaces []string
			if ns == "zdb" {
				zdbNamespaces, err = listZdbNamespaces(string(id))
				if err != nil {
					log.Err(err).Msg("failed to list container namespaces")
					continue
				}
			}

			// avoid to remove any used subvolume used by flistd for root container fs
			toSave[ctr.RootFS] = struct{}{}

			for _, mnt := range ctr.Mounts {
				// TODO: do we need this check ?
				// if !strings.HasPrefix(mnt.Source, "/mnt/") {
				// 	continue
				// }
				if len(zdbNamespaces) == 1 && zdbNamespaces[0] == "default" {
					err := common.DeleteZdbContainer(id, j.zbus)
					if err != nil {
						log.Err(err).Msgf("failed to delete zdb container %s", string(id))
						continue
					}
				} else {
					toSave[mnt.Source] = struct{}{}
				}
			}
		}
	}

	return toSave, nil
}

func socketDir(containerID string) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
}

func newZdbConnection(id string) (zdb.Client, error) {
	socket := fmt.Sprintf("unix://%s/zdb.sock", socketDir(id))
	cl := zdb.New(socket)
	return cl, cl.Connect()
}

func listZdbNamespaces(containterID string) ([]string, error) {
	zdbCl, err := newZdbConnection(containterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to 0-db")
	}

	defer zdbCl.Close()

	return zdbCl.Namespaces()
}
