package provision

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/stubs"
	"github.com/threefoldtech/zosv2/modules/zdb"
)

const (
	// TODO: make this configurable
	zdbFlistURL = "https://hub.grid.tf/tf-official-apps/threefoldtech-0-db-release-1.0.0.flist"
)

type ZDB struct {
	Size     uint64
	Mode     modules.ZDBMode
	Password string
	DiskType modules.DeviceType
	Public   bool
}

func ZdbProvision(ctx context.Context, reservation Reservation) (interface{}, error) {
	client := GetZBus(ctx)

	container := stubs.NewContainerModuleStub(client)
	flist := stubs.NewFlisterStub(client)
	storage := stubs.NewZDBModuleStub(client)
	identity := stubs.NewIdentityManagerStub(client)

	nodeID := identity.NodeID().Identity()
	nsID := reservation.ID

	var config ZDB
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return nil, errors.Wrap(err, "failed to decode reservation schema")
	}

	// TODO: verify if the namespace isn't already deployed

	vName, vPath, err := storage.Allocate(config.DiskType, config.Size*GiB, config.Mode)
	if err != nil {
		return nil, errors.Wrap(err, "failed to allocate storage")
	}

	// check if there is already a 0-DB running container on this volume
	c, err := container.Inspect(nodeID, modules.ContainerID(vName))
	if err != nil {
		log.Info().Msgf("0-db container %s not found, start creation", vName)

		log.Debug().Str("flist", zdbFlistURL).Msg("mounting flist")
		rootFS, err := flist.Mount(zdbFlistURL, "")
		if err != nil {
			return nil, err
		}

		id, err := container.Run(
			nodeID,
			modules.Container{
				Name:        vName,
				RootFS:      rootFS,
				Entrypoint:  "/bin/zdb",
				Interactive: false,
				// Network:     modules.NetworkInfo{Namespace: netNS}, TODO:
				Mounts: []modules.MountInfo{
					{
						Source:  vPath,
						Target:  "/data",
						Type:    "none",
						Options: []string{"bind"},
					},
				},
			})

		if err != nil {
			if err := flist.Umount(rootFS); err != nil {
				log.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
			}

			return nil, err
		}

		log.Info().Msgf("container created with id: '%s'", id)
	}

	// at this point there should always be a container running
	// for this 0-db
	c, err = container.Inspect(nodeID, modules.ContainerID(vName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to have a 0-db container running")
	}

	nss, err := zdb.Namespaces()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list namepace of 0-db %s", vName)
	}
	for _, ns := range nss {
		if ns == nsID {
			// namespace already exists
			return
		}
	}
	nsName := fmt.Sprintf("%s%d", vName, len(nss)+1)
	zdb.CreateNamespace()
}
