package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	cloudContainerFlist = "https://hub.grid.tf/tf-autobuilder/cloud-container-8730b6f.flist"
	cloudContainerName  = "cloud-container"
)

// ZMachine type
type ZMachine = zos.ZMachine

// ZMachineResult type
type ZMachineResult = zos.ZMachineResult

// FListInfo virtual machine details
type FListInfo struct {
	ImagePath string
}

func (t *FListInfo) IsContainer() bool {
	return len(t.ImagePath) == 0
}

var (
	_ provision.Manager     = (*Manager)(nil)
	_ provision.Initializer = (*Manager)(nil)
)

type Manager struct {
	zbus zbus.Client
}

func NewManager(zbus zbus.Client) *Manager {
	return &Manager{zbus}
}

func (m *Manager) Initialize(ctx context.Context) error {
	return m.initGPUs()
}

func (p *Manager) Provision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.virtualMachineProvisionImpl(ctx, wl)
}

func (p *Manager) vmMounts(ctx context.Context, deployment gridtypes.Deployment, mounts []zos.MachineMount, format bool, vm *pkg.VM) error {
	for _, mount := range mounts {
		wl, err := deployment.Get(mount.Name)
		if err != nil {
			return errors.Wrapf(err, "failed to get mount '%s' workload", mount.Name)
		}
		if wl.Result.State != gridtypes.StateOk {
			return fmt.Errorf("invalid disk '%s' state", mount.Name)
		}
		switch wl.Type {
		case zos.ZMountType:
			if err := p.mountDisk(ctx, wl, mount, format, vm); err != nil {
				return err
			}
		case zos.QuantumSafeFSType:
			if err := p.mountQsfs(wl, mount, vm); err != nil {
				return err
			}
		default:
			return fmt.Errorf("expecting a reservation of type '%s' or '%s' for disk '%s'", zos.ZMountType, zos.QuantumSafeFSType, mount.Name)
		}
	}
	return nil
}

func (p *Manager) mountDisk(ctx context.Context, wl *gridtypes.WorkloadWithID, mount zos.MachineMount, format bool, vm *pkg.VM) error {
	storage := stubs.NewStorageModuleStub(p.zbus)

	info, err := storage.DiskLookup(ctx, wl.ID.String())
	if err != nil {
		return errors.Wrapf(err, "failed to inspect disk '%s'", mount.Name)
	}

	if format {
		if err := storage.DiskFormat(ctx, wl.ID.String()); err != nil {
			return errors.Wrap(err, "failed to prepare mount")
		}
	}

	vm.Disks = append(vm.Disks, pkg.VMDisk{Path: info.Path, Target: mount.Mountpoint})

	return nil
}

func (p *Manager) mountQsfs(wl *gridtypes.WorkloadWithID, mount zos.MachineMount, vm *pkg.VM) error {

	var info zos.QuatumSafeFSResult
	if err := wl.Result.Unmarshal(&info); err != nil {
		return fmt.Errorf("invalid qsfs result '%s': %w", mount.Name, err)
	}
	wlID := wl.ID.String()
	vm.Shared = append(vm.Shared, pkg.SharedDir{ID: strings.ReplaceAll(wlID, "-", ""), Path: info.Path, Target: mount.Mountpoint})
	return nil
}

func (p *Manager) virtualMachineProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result zos.ZMachineResult, err error) {
	var (
		storage = stubs.NewStorageModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config ZMachine
	)
	if vm.Exists(ctx, wl.ID.String()) {
		return result, provision.ErrNoActionNeeded
	}

	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	if len(config.GPU) != 0 && !provision.IsRentedNode(ctx) {
		// you cannot use GPU unless this is a rented node
		return result, fmt.Errorf("usage of GPU is not allowed unless node is rented")
	}

	machine := pkg.VM{
		Name:   wl.ID.String(),
		CPU:    config.ComputeCapacity.CPU,
		Memory: config.ComputeCapacity.Memory,
	}

	// expand GPUs
	devices, err := p.expandGPUs(config.GPU)
	if err != nil {
		return result, errors.Wrap(err, "failed to prepare requested gpu device(s)")
	}

	for _, device := range devices {
		machine.Devices = append(machine.Devices, device.Slot)
	}

	// the config is validated by the engine. we now only support only one
	// private network
	if len(config.Network.Interfaces) != 1 {
		return result, fmt.Errorf("only one private network is support")
	}
	netConfig := config.Network.Interfaces[0]

	// check if public ipv4 is supported, should this be requested
	if !config.Network.PublicIP.IsEmpty() && !network.PublicIPv4Support(ctx) {
		return result, errors.New("public ipv4 is requested, but not supported on this node")
	}

	result.ID = wl.ID.String()
	result.IP = netConfig.IP.String()

	deployment, err := provision.GetDeployment(ctx)
	if err != nil {
		return result, errors.Wrap(err, "failed to get deployment")
	}
	networkInfo := pkg.VMNetworkInfo{
		Nameservers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("1.1.1.1"), net.ParseIP("2001:4860:4860::8888")},
	}

	var ifs []string
	var pubIf string

	defer func() {
		if err != nil {
			for _, nic := range ifs {
				_ = network.RemoveTap(ctx, nic)
			}
			if pubIf != "" {
				_ = network.DisconnectPubTap(ctx, pubIf)
			}
		}
	}()

	for _, nic := range config.Network.Interfaces {
		inf, err := p.newPrivNetworkInterface(ctx, deployment, wl, nic)
		if err != nil {
			return result, err
		}
		ifs = append(ifs, wl.ID.Unique(string(nic.Network)))
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
	}

	if !config.Network.PublicIP.IsEmpty() {
		// some public access is required, we need to add a public
		// interface to the machine with the right config.
		inf, err := p.newPubNetworkInterface(ctx, deployment, config)
		if err != nil {
			return result, err
		}

		ipWl, _ := deployment.Get(config.Network.PublicIP)
		pubIf = ipWl.ID.Unique("pub")
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
	}

	if config.Network.Planetary {
		inf, err := p.newYggNetworkInterface(ctx, wl)
		if err != nil {
			return result, err
		}

		log.Debug().Msgf("Planetary: %+v", inf)
		ifs = append(ifs, wl.ID.Unique("ygg"))
		networkInfo.Ifaces = append(networkInfo.Ifaces, inf)
		result.YggIP = inf.IPs[0].IP.String()
	}
	// - mount flist RO
	mnt, err := flist.Mount(ctx, wl.ID.String(), config.FList, pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
	}

	var imageInfo FListInfo
	// - detect type (container or VM)
	imageInfo, err = getFlistInfo(mnt)
	if err != nil {
		return result, err
	}

	log.Debug().Msgf("detected flist type: %+v", imageInfo)

	var (
		boot       pkg.Boot
		entrypoint string
		kernel     string
		initrd     string
	)

	// mount cloud-container flist (or reuse) which has kernel, initrd and also firmware
	hash, err := flist.FlistHash(ctx, cloudContainerFlist)
	if err != nil {
		return zos.ZMachineResult{}, errors.Wrap(err, "failed to get cloud-container flist hash")
	}

	// if the name changes (because flist changed, a new mount will be created)
	name := fmt.Sprintf("%s:%s", cloudContainerName, hash)
	// now mount cloud image also
	cloudImage, err := flist.Mount(ctx, name, cloudContainerFlist, pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrap(err, "failed to mount cloud container base image")
	}

	if imageInfo.IsContainer() {
		// - if Container, remount RW
		// prepare for container
		if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
			return result, errors.Wrapf(err, "failed to unmount flist: %s", wl.ID.String())
		}
		rootfsSize := config.RootSize()
		// create a persisted volume for the vm. we don't do it automatically
		// via the flist, so we have control over when to decomission this volume.
		// remounting in RW mode
		volName := fmt.Sprintf("rootfs:%s", wl.ID.String())
		volume, err := storage.VolumeCreate(ctx, volName, rootfsSize)
		if err != nil {
			return zos.ZMachineResult{}, errors.Wrap(err, "failed to create vm rootfs")
		}

		defer func() {
			if err != nil {
				// vm creation failed,
				if err := storage.VolumeDelete(ctx, volName); err != nil {
					log.Error().Err(err).Str("volume", volName).Msg("failed to delete persisted volume")
				}
			}
		}()

		mnt, err = flist.Mount(ctx, wl.ID.String(), config.FList, pkg.MountOptions{
			ReadOnly:        false,
			PersistedVolume: volume.Path,
		})

		if err != nil {
			return result, errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
		}

		// inject container kernel and init
		kernel = filepath.Join(cloudImage, "kernel")
		initrd = filepath.Join(cloudImage, "initramfs-linux.img")

		boot = pkg.Boot{
			Type: pkg.BootVirtioFS,
			Path: mnt,
		}

		if err := fListStartup(&config, filepath.Join(mnt, ".startup.toml")); err != nil {
			return result, errors.Wrap(err, "failed to apply startup config from flist")
		}

		entrypoint = config.Entrypoint
		if err := p.vmMounts(ctx, deployment, config.Mounts, true, &machine); err != nil {
			return result, err
		}
		if config.Corex {
			if err := p.copyFile("/usr/bin/corex", filepath.Join(mnt, "corex"), 0755); err != nil {
				return result, errors.Wrap(err, "failed to inject corex binary")
			}
			entrypoint = "/corex --ipv6 -d 7 --interface eth0"
		}
	} else {
		// if a VM the vm has to have at least one mount
		if len(config.Mounts) == 0 {
			err = fmt.Errorf("at least one mount has to be attached for Vm mode")
			return result, err
		}

		kernel = filepath.Join(cloudImage, "hypervisor-fw")
		var disk *gridtypes.WorkloadWithID
		disk, err = deployment.Get(config.Mounts[0].Name)
		if err != nil {
			return result, err
		}

		if disk.Type != zos.ZMountType {
			return result, fmt.Errorf("mount is not not a valid disk workload")
		}

		if disk.Result.State != gridtypes.StateOk {
			return result, fmt.Errorf("boot disk was not deployed correctly")
		}
		var info pkg.VDisk
		info, err = storage.DiskLookup(ctx, disk.ID.String())
		if err != nil {
			return result, errors.Wrap(err, "disk does not exist")
		}

		//TODO: DiskWrite will not override the disk if it already has a partition table
		// or a filesystem. this means that if later the disk is assigned to a new VM with
		// a different flist it will have the same old operating system copied from previous
		// setup.
		if err = storage.DiskWrite(ctx, disk.ID.String(), imageInfo.ImagePath); err != nil {
			return result, errors.Wrap(err, "failed to write image to disk")
		}

		boot = pkg.Boot{
			Type: pkg.BootDisk,
			Path: info.Path,
		}
		if err := p.vmMounts(ctx, deployment, config.Mounts[1:], false, &machine); err != nil {
			return result, err
		}
	}

	// - Attach mounts
	// - boot
	machine.Network = networkInfo
	machine.KernelImage = kernel
	machine.InitrdImage = initrd
	machine.Boot = boot
	machine.Entrypoint = entrypoint
	machine.Environment = config.Env
	machine.Hostname = wl.Name.String()

	machineInfo, err := vm.Run(ctx, machine)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		log.Error().Err(err).Msg("cleaning up vm deployment duo to an error")
		_ = vm.Delete(ctx, wl.ID.String())
	}
	result.ConsoleURL = machineInfo.ConsoleURL
	return result, err
}
func (p *Manager) copyFile(srcPath string, destPath string, permissions os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrapf(err, "Coludn't find %s on the node", srcPath)
	}
	defer src.Close()
	dest, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, permissions)
	if err != nil {
		return errors.Wrapf(err, "Coludn't create %s file", destPath)
	}
	defer dest.Close()
	_, err = io.Copy(dest, src)
	if err != nil {
		return errors.Wrapf(err, "Couldn't copy to %s", destPath)
	}
	return nil
}

func (p *Manager) Deprovision(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	var (
		flist   = stubs.NewFlisterStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)
		storage = stubs.NewStorageModuleStub(p.zbus)

		cfg ZMachine
	)

	if err := json.Unmarshal(wl.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if _, err := vm.Inspect(ctx, wl.ID.String()); err == nil {
		if err := vm.Delete(ctx, wl.ID.String()); err != nil {
			return errors.Wrapf(err, "failed to delete vm %s", wl.ID)
		}
	}

	if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
		log.Error().Err(err).Msg("failed to unmount machine flist")
	}

	volName := fmt.Sprintf("rootfs:%s", wl.ID.String())
	if err := storage.VolumeDelete(ctx, volName); err != nil {
		log.Error().Err(err).Str("name", volName).Msg("failed to delete rootfs volume")
	}

	for _, inf := range cfg.Network.Interfaces {
		tapName := wl.ID.Unique(string(inf.Network))

		if err := network.RemoveTap(ctx, tapName); err != nil {
			return errors.Wrap(err, "could not clean up tap device")
		}
	}

	if cfg.Network.Planetary {
		tapName := wl.ID.Unique("ygg")
		if err := network.RemoveTap(ctx, tapName); err != nil {
			return errors.Wrap(err, "could not clean up tap device")
		}
	}

	if len(cfg.Network.PublicIP) > 0 {
		// TODO: we need to make sure workload status reflects the actual status by the engine
		// this is not the case anymore.
		ipWl, err := provision.GetWorkload(ctx, cfg.Network.PublicIP)
		if err != nil {
			return err
		}
		ifName := ipWl.ID.Unique("pub")
		if err := network.RemovePubTap(ctx, ifName); err != nil {
			return errors.Wrap(err, "could not clean up public tap device")
		}
	}

	return nil
}
