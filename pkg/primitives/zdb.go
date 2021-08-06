package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/zdb"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	nwmod "github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	// https://hub.grid.tf/api/flist/tf-autobuilder/threefoldtech-0-db-development.flist/light
	// To get the latest symlink pointer
	zdbFlistURL         = "https://hub.grid.tf/tf-autobuilder/threefoldtech-0-db-development-b5155357d5.flist"
	zdbContainerNS      = "zdb"
	zdbContainerDataMnt = "/zdb"
	zdbPort             = 9900
)

// ZDB types
type ZDB = zos.ZDB

type ZDBContainer pkg.Container

func (z *ZDBContainer) DataMount() (string, error) {
	for _, mnt := range z.Mounts {
		if mnt.Target == zdbContainerDataMnt {
			return mnt.Source, nil
		}
	}

	return "", fmt.Errorf("container '%s' does not have a valid data mount", z.Name)
}

func (p *Primitives) zdbProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.zdbProvisionImpl(ctx, wl)
}

func (p *Primitives) zdbListContainers(ctx context.Context) (map[pkg.ContainerID]ZDBContainer, error) {
	var (
		contmod = stubs.NewContainerModuleStub(p.zbus)
	)

	containerIDs, err := contmod.List(ctx, zdbContainerNS)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list running containers")
	}

	// for each container we try to find a free space to jam in this new zdb namespace
	// request
	m := make(map[pkg.ContainerID]ZDBContainer)

	for _, containerID := range containerIDs {
		container, err := contmod.Inspect(ctx, zdbContainerNS, containerID)
		if err != nil {
			log.Error().Err(err).Str("container-id", string(containerID)).Msg("failed to inspect zdb container")
			continue
		}
		cont := ZDBContainer(container)

		if _, err = cont.DataMount(); err != nil {
			log.Error().Err(err).Msg("failed to get data directory of zdb container")
			continue
		}
		m[containerID] = cont
	}

	return m, nil
}

func (p *Primitives) zdbProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (zos.ZDBResult, error) {
	var (
		contmod = stubs.NewContainerModuleStub(p.zbus)
		nsID    = wl.ID.String()
		config  ZDB
	)
	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to decode reservation schema")
	}

	ipsToString := func(ips []net.IP) []string {
		result := make([]string, 0, len(ips))
		for _, ip := range ips {
			result = append(result, ip.String())
		}

		return result
	}
	// for each container we try to find a free space to jam in this new zdb namespace
	// request
	containers, err := p.zdbListContainers(ctx)
	if err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to list container data volumes")
	}

	// check if namespace already exist
	for id, container := range containers {
		dataPath, _ := container.DataMount() // the error should not happen

		index := zdb.NewIndex(dataPath)
		nss, err := index.Namespaces()
		if err != nil {
			// skip or error
			log.Error().Err(err).Str("container-id", string(id)).Msg("couldn't list namespaces")
			continue
		}

		for _, ns := range nss {
			if ns.Name != nsID {
				continue
			}

			containerIPs, err := p.waitZDBIPs(ctx, container.Network.Namespace)
			if err != nil {
				return zos.ZDBResult{}, errors.Wrap(err, "failed to find IP address on zdb0 interface")
			}

			return zos.ZDBResult{
				Namespace: nsID,
				IPs:       ipsToString(containerIPs),
				Port:      zdbPort,
			}, nil
		}
	}

	// otherwise, we check if we can fit this namespace into  one of the running zdbs.

	// if we reached here, we need to create the 0-db namespace
	log.Debug().Msg("allocating storage for namespace")
	allocation, err := storage.Allocate(ctx, nsID, config.DiskType, config.Size, config.Mode)
	if err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to allocate storage")
	}

	containerID := pkg.ContainerID(allocation.VolumeID)

	cont, err := p.ensureZdbContainer(ctx, allocation)
	if err != nil {
		return zos.ZDBResult{}, errors.Wrapf(err, "failed to ensure zdb containe running")
	}

	containerIPs, err := p.waitZDBIPs(ctx, cont.Network.Namespace)
	if err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to find IP address on zdb0 interface")
	}
	log.Warn().Msgf("ip for zdb containers %s", containerIPs)

	// this call will actually configure the namespace in zdb and set the password
	if err := p.createZDBNamespace(containerID, nsID, config); err != nil {
		return zos.ZDBResult{}, errors.Wrap(err, "failed to create zdb namespace")
	}

	return zos.ZDBResult{
		Namespace: nsID,
		IPs: func() []string {
			ips := make([]string, len(containerIPs))
			for i, ip := range containerIPs {
				ips[i] = ip.String()
			}
			return ips
		}(),
		Port: zdbPort,
	}, nil
}

func (p *Primitives) ensureZdbContainer(ctx context.Context, device pkg.Device) (pkg.Container, error) {
	var container = stubs.NewContainerModuleStub(p.zbus)

	name := pkg.ContainerID(device.ID)

	cont, err := container.Inspect(ctx, zdbContainerNS, name)
	if err != nil && strings.Contains(err.Error(), "not found") {
		// container not found, create one
		if err := p.createZdbContainer(ctx, device); err != nil {
			return cont, err
		}
		cont, err = container.Inspect(ctx, zdbContainerNS, name)
		if err != nil {
			return pkg.Container{}, err
		}
	} else if err != nil {
		// other error
		return pkg.Container{}, err
	}

	return cont, nil

}

func (p *Primitives) zdbRootFS(ctx context.Context) (string, error) {
	var flist = stubs.NewFlisterStub(p.zbus)
	var err error
	var rootFS string

	rootFS, err = flist.Mount(ctx, zdbFlistURL, "", pkg.MountOptions{
		Limit:    10 * gridtypes.Megabyte,
		ReadOnly: false,
	})

	if err != nil {
		log.Error().Err(err).Msgf("failed to allocate rootfs for zdb container: %s", err)
	}

	return rootFS, nil
}

func (p *Primitives) createZdbContainer(ctx context.Context, device pkg.Device) error {
	var (
		name       = pkg.ContainerID(device.ID)
		cont       = stubs.NewContainerModuleStub(p.zbus)
		flist      = stubs.NewFlisterStub(p.zbus)
		volumePath = device.Path
		network    = stubs.NewNetworkerStub(p.zbus)

		slog = log.With().Str("containerID", string(name)).Logger()
	)

	slog.Debug().Str("flist", zdbFlistURL).Msg("mounting flist")

	rootFS, err := p.zdbRootFS(ctx)
	if err != nil {
		return err
	}

	cleanup := func() {
		if err := cont.Delete(ctx, zdbContainerNS, name); err != nil {
			slog.Error().Err(err).Msg("failed to delete 0-db container")
		}

		if err := flist.Unmount(ctx, string(name)); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}
	}

	// create the network namespace and macvlan for the 0-db container
	netNsName, err := network.ZDBPrepare(ctx, device.ID)
	if err != nil {
		if err := flist.Unmount(ctx, string(name)); err != nil {
			slog.Error().Err(err).Str("path", rootFS).Msgf("failed to unmount")
		}

		return errors.Wrap(err, "failed to prepare zdb network")
	}

	socketDir := socketDir(name)
	if err := os.MkdirAll(socketDir, 0550); err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "failed to create directory: %s", socketDir)
	}

	cl := zdbConnection(name)
	if err := cl.Connect(); err == nil {
		// it seems there is a running container already
		cl.Close()
		return nil
	}

	// make sure the file does not exist otherwise we get the address already in use error
	if err := os.Remove(socketFile(name)); err != nil && !os.IsNotExist(err) {
		return err
	}

	cmd := fmt.Sprintf("/bin/zdb --data /zdb/data --index /zdb/index  --listen :: --port %d --socket /socket/zdb.sock --dualnet", zdbPort)

	err = p.zdbRun(ctx, string(name), rootFS, cmd, netNsName, volumePath, socketDir)
	if err != nil {
		cleanup()
		return errors.Wrap(err, "failed to create container")
	}

	cl = zdbConnection(name)
	defer cl.Close()

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Second * 20
	bo.MaxElapsedTime = time.Minute * 2

	if err := backoff.RetryNotify(cl.Connect, bo, func(err error, d time.Duration) {
		log.Debug().Err(err).Str("duration", d.String()).Msg("waiting for zdb to start")
	}); err != nil {
		cleanup()
		return errors.Wrapf(err, "failed to establish connection to zdb")
	}

	return nil
}

func (p *Primitives) zdbRun(ctx context.Context, name string, rootfs string, cmd string, netns string, volumepath string, socketdir string) error {
	var cont = stubs.NewContainerModuleStub(p.zbus)

	_, err := cont.Run(
		ctx,
		zdbContainerNS,
		pkg.Container{
			Name:        name,
			RootFS:      rootfs,
			Entrypoint:  cmd,
			Interactive: false,
			Network:     pkg.NetworkInfo{Namespace: netns},
			Mounts: []pkg.MountInfo{
				{
					Source: volumepath,
					Target: zdbContainerDataMnt,
				},
				{
					Source: socketdir,
					Target: "/socket",
				},
			},
		})

	return err
}

func (p *Primitives) waitZDBIPs(ctx context.Context, namespace string) ([]net.IP, error) {
	// TODO: this method need to be abstracted, since it's now depends on the knewledge
	// of the networking daemon internal (interfaces names)
	// may be at least just get all ips from all interfaces inside the namespace
	// will be a slightly better solution
	var (
		network      = stubs.NewNetworkerStub(p.zbus)
		containerIPs []net.IP
	)

	getIP := func() error {

		ips, err := network.Addrs(ctx, nwmod.ZDBPubIface, namespace)
		if err != nil {
			log.Debug().Err(err).Msg("not ip public found, waiting")
			return err
		}
		yggIps, err := network.Addrs(ctx, nwmod.ZDBYggIface, namespace)
		if err != nil {
			return err
		}

		ips = append(ips, yggIps...)

		var (
			public = false
			ygg    = false
		)
		containerIPs = containerIPs[:0]

		for _, ip := range ips {
			if isPublic(ip) && !isYgg(ip) {
				log.Warn().IPAddr("ip", ip).Msg("0-db container public ip found")
				public = true
				containerIPs = append(containerIPs, ip)
			}
			if isYgg(ip) {
				log.Warn().IPAddr("ip", ip).Msg("0-db container ygg ip found")
				ygg = true
				containerIPs = append(containerIPs, ip)
			}
		}

		log.Warn().Msgf("public %v ygg: %v", public, ygg)
		if public && ygg {
			return nil
		}
		return fmt.Errorf("waiting for more addresses")
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Minute
	bo.MaxElapsedTime = time.Minute * 2

	if err := backoff.RetryNotify(getIP, bo, func(err error, d time.Duration) {
		log.Debug().Err(err).Str("duration", d.String()).Msg("failed to get zdb public IP")
	}); err != nil && len(containerIPs) == 0 {
		return nil, errors.Wrapf(err, "failed to get an IP for interface")
	}

	return containerIPs, nil
}

func (p *Primitives) createZDBNamespace(containerID pkg.ContainerID, nsID string, config ZDB) error {
	zdbCl := zdbConnection(containerID)
	defer zdbCl.Close()
	if err := zdbCl.Connect(); err != nil {
		return errors.Wrapf(err, "failed to connect to 0-db: %s", containerID)
	}

	exists, err := zdbCl.Exist(nsID)
	if err != nil {
		return err
	}
	if !exists {
		if err := zdbCl.CreateNamespace(nsID); err != nil {
			return errors.Wrapf(err, "failed to create namespace in 0-db: %s", containerID)
		}
	}

	if config.Password != "" {
		if err := zdbCl.NamespaceSetPassword(nsID, config.Password); err != nil {
			return errors.Wrapf(err, "failed to set password namespace %s in 0-db: %s", nsID, containerID)
		}
	}

	if err := zdbCl.NamespaceSetPublic(nsID, config.Public); err != nil {
		return errors.Wrapf(err, "failed to make namespace %s public in 0-db: %s", nsID, containerID)
	}

	if err := zdbCl.NamespaceSetSize(nsID, uint64(config.Size)); err != nil {
		return errors.Wrapf(err, "failed to set size on namespace %s in 0-db: %s", nsID, containerID)
	}

	return nil
}

func (p *Primitives) zdbDecommission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	return fmt.Errorf("not implemented")

	// var (
	// 	storage       = stubs.NewStorageModuleStub(p.zbus)
	// 	storageClient = stubs.NewStorageModuleStub(p.zbus)

	// 	config zos.ZDB
	// 	nsID   = wl.ID.String()
	// )

	// if err := json.Unmarshal(wl.Data, &config); err != nil {
	// 	return errors.Wrap(err, "failed to decode reservation schema")
	// }

	// device, err := storage.Find(ctx, wl.ID.String())
	// if err != nil && strings.Contains(err.Error(), "not found") {
	// 	return nil
	// } else if err != nil {
	// 	return err
	// }

	// _, err = p.ensureZdbContainer(ctx, device)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to find namespace zdb container")
	// }

	// containerID := pkg.ContainerID(device.VolumeID)

	// zdbCl := zdbConnection(containerID)
	// defer zdbCl.Close()
	// if err := zdbCl.Connect(); err != nil {
	// 	return errors.Wrapf(err, "failed to connect to 0-db: %s", containerID)
	// }

	// if err := zdbCl.DeleteNamespace(nsID); err != nil {
	// 	return errors.Wrapf(err, "failed to delete namespace in 0-db: %s", containerID)
	// }

	// ns, err := zdbCl.Namespaces()
	// if err != nil {
	// 	return errors.Wrap(err, "failed to retrieve zdb namespaces")
	// }

	// log.Info().Msgf("zdb has %d namespaces left", len(ns))

	// // If there are no more namespaces left except for the default namespace, we can delete this subvolume
	// if len(ns) == 1 && ns[0] == "default" {
	// 	log.Info().Msg("decommissioning zdb container because there are no more namespaces left")
	// 	err = p.deleteZdbContainer(ctx, containerID)
	// 	if err != nil {
	// 		return errors.Wrap(err, "failed to decommission zdb container")
	// 	}

	// 	log.Info().Msgf("deleting subvolumes of reservation: %s", device.VolumeID)
	// 	// we also need to delete the flist volume
	// 	return storageClient.ReleaseFilesystem(ctx, device.VolumeID)
	// }

	// return nil
}

func (p *Primitives) deleteZdbContainer(ctx context.Context, containerID pkg.ContainerID) error {
	container := stubs.NewContainerModuleStub(p.zbus)
	flist := stubs.NewFlisterStub(p.zbus)

	info, err := container.Inspect(ctx, "zdb", containerID)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to inspect container '%s'", containerID)
	}

	if err := container.Delete(ctx, "zdb", containerID); err != nil {
		return errors.Wrapf(err, "failed to delete container %s", containerID)
	}

	network := stubs.NewNetworkerStub(p.zbus)
	if err := network.ZDBDestroy(ctx, info.Network.Namespace); err != nil {
		return errors.Wrapf(err, "failed to destroy zdb network namespace")
	}

	if err := flist.Unmount(ctx, string(containerID)); err != nil {
		return errors.Wrapf(err, "failed to unmount flist at %s", info.RootFS)
	}

	return nil
}

func socketDir(containerID pkg.ContainerID) string {
	return fmt.Sprintf("/var/run/zdb_%s", containerID)
}

func socketFile(containerID pkg.ContainerID) string {
	return filepath.Join(socketDir(containerID), "zdb.sock")
}

// we declare this method as a variable so we can
// mock it in testing.
var zdbConnection = func(id pkg.ContainerID) zdb.Client {
	socket := fmt.Sprintf("unix://%s", socketFile(id))
	return zdb.New(socket)
}

// isPublic check if ip is a IPv6 public address
func isPublic(ip net.IP) bool {
	if ip.To4() != nil {
		return false
	}

	return !(ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast())
}

// isPublic check if ip is a part of the yggdrasil 200::/7 range
var yggNet = net.IPNet{
	IP:   net.ParseIP("200::"),
	Mask: net.CIDRMask(7, 128),
}

func isYgg(ip net.IP) bool {
	return yggNet.Contains(ip)
}

func (p *Primitives) InitializeZDB(ctx context.Context) error {
	var (
		storage  = stubs.NewStorageModuleStub(p.zbus)
		contmod  = stubs.NewContainerModuleStub(p.zbus)
		flistmod = stubs.NewFlisterStub(p.zbus)
	)
	// fetching extected hash
	log.Debug().Msg("fetching flist hash")
	expected, err := flistmod.FlistHash(ctx, zdbFlistURL)
	if err != nil {
		log.Error().Err(err).Msg("could not load expected flist hash")
		return err
	}

	devices, err := storage.Devices(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to list allocated zdb devices")
	}

	poolNames := make(map[string]pkg.Device)
	for _, device := range devices {
		poolNames[device.ID] = device
	}

	containers, err := contmod.List(ctx, zdbContainerNS)
	if err != nil {
		return errors.Wrap(err, "failed to list running zdb container")
	}

	for _, container := range containers {
		if err := p.upgradeRuntime(ctx, expected, container); err != nil {
			log.Error().Err(err).Msg("failed to upgrade running zdb container")
		}

		delete(poolNames, string(container))
	}

	// do we still have allocated pools that does not have associated zdbs.
	for _, device := range poolNames {
		if _, err := p.ensureZdbContainer(ctx, device); err != nil {
			log.Error().Err(err).Str("pool", device.ID).Msg("failed to create zdb container associated with pool")
		}
	}
	return nil
}

func (p *Primitives) upgradeRuntime(ctx context.Context, expected string, container pkg.ContainerID) error {
	var (
		flistmod = stubs.NewFlisterStub(p.zbus)
		contmod  = stubs.NewContainerModuleStub(p.zbus)
	)
	continfo, err := contmod.Inspect(ctx, zdbContainerNS, container)

	if err != nil {
		return err
	}

	hash, err := flistmod.HashFromRootPath(ctx, continfo.RootFS)
	if err != nil {
		return errors.Wrap(err, "could not find container running flist hash")
	}

	log.Debug().Str("hash", hash).Msg("running container hash")
	if hash == expected {
		return nil
	}

	log.Info().Str("id", string(container)).Msg("restarting container, update found")

	// extracting required informations
	volumeid := continfo.Name // VolumeID is the Container Name
	volumepath := ""          // VolumePath is /data mount on the container
	socketdir := ""           // SocketDir is /socket on the container
	zdbcmd := continfo.Entrypoint
	netns := continfo.Network.Namespace

	log.Info().Str("id", volumeid).Str("path", volumepath).Msg("rebuild zdb container")

	for _, mnt := range continfo.Mounts {
		if mnt.Target == zdbContainerDataMnt {
			volumepath = mnt.Source
		}

		if mnt.Target == "/socket" {
			socketdir = mnt.Source
		}
	}

	if volumepath == "" {
		return fmt.Errorf("could not grab container /data mountpoint")
	}

	// stopping running zdb

	if err := contmod.Delete(ctx, zdbContainerNS, container); err != nil {
		return errors.Wrap(err, "could not stop running zdb container")
	}

	// cleanup old containers rootfs
	if err = flistmod.Unmount(ctx, volumeid); err != nil {
		log.Error().Err(err).Str("path", continfo.RootFS).Msgf("failed to unmount old zdb container")
	}

	// restarting zdb

	// mount the new flist
	rootfs, err := p.zdbRootFS(ctx)
	if err != nil {
		return errors.Wrap(err, "could not initialize zdb rootfs")
	}

	// respawn the container
	err = p.zdbRun(ctx, volumeid, rootfs, zdbcmd, netns, volumepath, socketdir)
	if err != nil {
		log.Error().Err(err).Msg("could not restart zdb container")

		if err = flistmod.Unmount(ctx, volumeid); err != nil {
			log.Error().Err(err).Str("path", rootfs).Msgf("failed to unmount zdb container")
		}
	}

	return nil
}
