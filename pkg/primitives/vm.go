package primitives

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	// this probably need a way to update. for now just hard code it
	cloudContainerFlist = "https://hub.grid.tf/azmy.3bot/cloud-container.flist"
	cloudContainerName  = "cloud-container"
)

// VirtualMachine type
type VirtualMachine = zos.VirtualMachine

// VirtualMachineResult type
type VirtualMachineResult = zos.VirtualMachineResult

// FListInfo virtual machine details
type FListInfo struct {
	Container bool
	Initrd    string
	Kernel    string
	ImagePath string
}

func (p *Primitives) virtualMachineProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.virtualMachineProvisionImpl(ctx, wl)
}

func (p *Primitives) mountsToDisks(ctx context.Context, deployment gridtypes.Deployment, disks []zos.MachineMount, format bool) ([]pkg.VMDisk, error) {
	storage := stubs.NewVDiskModuleStub(p.zbus)

	var results []pkg.VMDisk
	for _, disk := range disks {
		wl, err := deployment.Get(disk.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get disk '%s' workload", disk.Name)
		}
		if wl.Type != zos.ZMountType {
			return nil, fmt.Errorf("expecting a reservation of type '%s' for disk '%s'", zos.ZMountType, disk.Name)
		}
		if wl.Result.State != gridtypes.StateOk {
			return nil, fmt.Errorf("invalid disk '%s' state", disk.Name)
		}

		info, err := storage.Inspect(ctx, wl.ID.String())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to inspect disk '%s'", disk.Name)
		}

		if format {
			if err := storage.EnsureFilesystem(ctx, wl.ID.String()); err != nil {
				return nil, errors.Wrap(err, "failed to prepare mount")
			}
		}

		results = append(results, pkg.VMDisk{Path: info.Path, Target: disk.Mountpoint})
	}

	return results, nil
}
func (p *Primitives) virtualMachineProvisionImpl(ctx context.Context, wl *gridtypes.WorkloadWithID) (result KubernetesResult, err error) {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config VirtualMachine
	)

	if vm.Exists(ctx, wl.ID.String()) {
		return result, provision.ErrDidNotChange
	}

	if err := json.Unmarshal(wl.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}
	// Should config.Vaid() be called here?

	deployment := provision.GetDeployment(ctx)

	// the config is validated by the engine. we now only support only one
	// private network
	netConfig := config.Network.Interfaces[0]
	netID := zos.NetworkID(deployment.TwinID, netConfig.Network)

	// hash to avoid tapName > 16 errors
	tapName := hashDeployment(wl.ID)

	exists, err := network.TapExists(ctx, tapName)
	if err != nil {
		return result, errors.Wrap(err, "could not check if tap device exists")
	}

	if exists {
		return result, errors.New("found a tap device with the same name. shouldn't happen")
	}

	// check if public ipv4 is supported, should this be requested
	if !config.Network.PublicIP.IsEmpty() && !network.PublicIPv4Support(ctx) {
		return result, errors.New("public ipv4 is requested, but not supported on this node")
	}

	result.ID = wl.ID.String()
	result.IP = netConfig.IP.String()

	// - mount flist RO
	mnt, err := flist.Mount(ctx, wl.ID.String(), config.FList, pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
	}

	// defer func() {
	// 	if err != nil {
	// 		flist.Unmount(ctx, wl.ID.String())
	// 	}
	// }()

	var imageInfo FListInfo
	// - detect type (container or VM)
	imageInfo, err = getFlistInfo(mnt)
	if err != nil {
		return result, err
	}

	log.Debug().Msgf("detected flist type: %+v", imageInfo)

	var boot pkg.Boot
	var disks []pkg.VMDisk

	// "root=/dev/vda rw console=ttyS0 reboot=k panic=1"
	cmd := pkg.KernelArgs{
		"rw":      "",
		"console": "ttyS0",
		"reboot":  "k",
		"panic":   "1",
		"root":    "/dev/vda",
	}

	if imageInfo.Container {
		// - if Container, remount RW
		// prepare for container
		if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
			return result, errors.Wrapf(err, "failed to unmount flist: %s", wl.ID.String())
		}
		// remounting in RW mode
		mnt, err = flist.Mount(ctx, wl.ID.String(), config.FList, pkg.DefaultMountOptions)
		if err != nil {
			return result, errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
		}

		// now mount cloud image also
		cloudImage, err := flist.Mount(ctx, cloudContainerName, cloudContainerFlist, pkg.ReadOnlyMountOptions)
		if err != nil {
			return result, errors.Wrap(err, "failed to mount cloud container base image")
		}
		// inject container kernel and init
		imageInfo.Kernel = filepath.Join(cloudImage, "kernel")
		imageInfo.Initrd = filepath.Join(cloudImage, "initramfs-linux.img")

		boot = pkg.Boot{
			Type: pkg.BootVirtioFS,
			Path: mnt,
		}
		// change the root boot to use the right virtiofs tag
		cmd["init"] = config.Entrypoint

		disks, err = p.mountsToDisks(ctx, deployment, config.Mounts, true)
		if err != nil {
			return result, err
		}
	} else {
		// if a VM the vm has to have at least one mount
		if len(config.Mounts) == 0 {
			err = fmt.Errorf("at least one mount has to be attached for Vm mode")
			return result, err
		}

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
		info, err = storage.Inspect(ctx, disk.ID.String())
		if err != nil {
			return result, errors.Wrap(err, "disk does not exist")
		}

		//TODO: this should not happen if disk image was written before !!
		// fs detection must be done here
		if err = storage.WriteImage(ctx, disk.ID.String(), imageInfo.ImagePath); err != nil {
			return result, errors.Wrap(err, "failed to write image to disk")
		}

		boot = pkg.Boot{
			Type: pkg.BootDisk,
			Path: info.Path,
		}
		// we don't format disks attached to VMs, it's up to the vm to decide that
		disks, err = p.mountsToDisks(ctx, deployment, config.Mounts[1:], false)
		if err != nil {
			return result, err
		}
	}

	// - Attach mounts
	// - boot

	var iface string
	iface, err = network.SetupTap(ctx, netID, tapName)
	if err != nil {
		return result, errors.Wrap(err, "could not set up tap device")
	}

	defer func() {
		if err != nil {
			_ = network.RemoveTap(ctx, tapName)
		}
	}()

	var pubIface string
	if len(config.Network.PublicIP) > -0 {
		ipWl, err := deployment.Get(config.Network.PublicIP)
		if err != nil {
			return zos.KubernetesResult{}, err
		}
		name := ipWl.ID.String()
		pubIface, err = network.SetupPubTap(ctx, name)
		if err != nil {
			return result, errors.Wrap(err, "could not set up tap device for public network")
		}

		defer func() {
			if err != nil {
				_ = network.RemovePubTap(ctx, name)
			}
		}()
	}

	var netInfo pkg.VMNetworkInfo
	netInfo, err = p.buildNetworkInfo(ctx, deployment, iface, pubIface, config)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}

	err = p.vmRun(ctx, wl.ID.String(), &config, boot, disks, imageInfo, cmd, netInfo)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		vm.Delete(ctx, wl.ID.String())
	}

	return result, err
}

func (p *Primitives) vmDecomission(ctx context.Context, wl *gridtypes.WorkloadWithID) error {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		cfg VirtualMachine
	)

	if err := json.Unmarshal(wl.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if _, err := vm.Inspect(ctx, wl.ID.String()); err == nil {
		if err := vm.Delete(ctx, wl.ID.String()); err != nil {
			return errors.Wrapf(err, "failed to delete vm %s", wl.ID)
		}
	}

	tapName := hashDeployment(wl.ID)

	if err := network.RemoveTap(ctx, tapName); err != nil {
		return errors.Wrap(err, "could not clean up tap device")
	}

	if len(cfg.Network.PublicIP) > 0 {
		deployment := provision.GetDeployment(ctx)
		ipWl, err := deployment.Get(cfg.Network.PublicIP)
		ifName := ipWl.ID.String()
		if err != nil {
			return err
		}
		if err := network.RemovePubTap(ctx, ifName); err != nil {
			return errors.Wrap(err, "could not clean up public tap device")
		}
	}

	if err := storage.Deallocate(ctx, fmt.Sprintf("%s-%s", wl.ID, "vda")); err != nil {
		return errors.Wrap(err, "could not remove vDisk")
	}

	return nil
}

func (p *Primitives) vmRun(
	ctx context.Context,
	name string,
	config *VirtualMachine,
	boot pkg.Boot,
	disks []pkg.VMDisk,
	imageInfo FListInfo,
	cmdline pkg.KernelArgs,
	networkInfo pkg.VMNetworkInfo) error {

	vm := stubs.NewVMModuleStub(p.zbus)

	cap := config.ComputeCapacity
	// installed disk
	kubevm := pkg.VM{
		Name:        name,
		CPU:         cap.CPU,
		Memory:      cap.Memory,
		Network:     networkInfo,
		KernelImage: imageInfo.Kernel,
		InitrdImage: imageInfo.Initrd,
		KernelArgs:  cmdline,
		Boot:        boot,
		Environment: config.Env,
		Disks:       disks,
	}

	return vm.Run(ctx, kubevm)
}

func hashDeployment(wid gridtypes.WorkloadID) string {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprint(wid))
	h := md5.Sum(buf.Bytes())
	b := base58.Encode(h[:])
	if len(b) > 13 {
		b = b[:13]
	}
	return string(b)
}
