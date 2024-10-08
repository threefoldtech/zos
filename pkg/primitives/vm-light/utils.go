package vmlight

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos4/pkg"
	"github.com/threefoldtech/zos4/pkg/gridtypes"
	"github.com/threefoldtech/zos4/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos4/pkg/stubs"
)

// fill up the VM (machine) object with write boot config for a full virtual machine (with a disk image)
func (p *Manager) prepVirtualMachine(
	ctx context.Context,
	cloudImage string,
	imageInfo FListInfo,
	machine *pkg.VM,
	config *zos.ZMachineLight,
	deployment *gridtypes.Deployment,
	wl *gridtypes.WorkloadWithID,
) error {
	storage := stubs.NewStorageModuleStub(p.zbus)
	// if a VM the vm has to have at least one mount
	if len(config.Mounts) == 0 {
		return fmt.Errorf("at least one mount has to be attached for Vm mode")
	}

	machine.KernelImage = filepath.Join(cloudImage, "hypervisor-fw")
	disk, err := deployment.Get(config.Mounts[0].Name)
	if err != nil {
		return err
	}

	if disk.Type != zos.ZMountType {
		return fmt.Errorf("mount is not a valid disk workload")
	}

	if disk.Result.State != gridtypes.StateOk {
		return fmt.Errorf("boot disk was not deployed correctly")
	}

	info, err := storage.DiskLookup(ctx, disk.ID.String())
	if err != nil {
		return errors.Wrap(err, "disk does not exist")
	}

	// TODO: DiskWrite will not override the disk if it already has a partition table
	// or a filesystem. this means that if later the disk is assigned to a new VM with
	// a different flist it will have the same old operating system copied from previous
	// setup.
	if err = storage.DiskWrite(ctx, disk.ID.String(), imageInfo.ImagePath); err != nil {
		return errors.Wrap(err, "failed to write image to disk")
	}

	machine.Boot = pkg.Boot{
		Type: pkg.BootDisk,
		Path: info.Path,
	}

	return p.vmMounts(ctx, deployment, config.Mounts[1:], false, machine)
}

// prepare the machine and fill it up with proper boot flags for a container VM
func (p *Manager) prepContainer(
	ctx context.Context,
	cloudImage string,
	imageInfo FListInfo,
	machine *pkg.VM,
	config *zos.ZMachineLight,
	deployment *gridtypes.Deployment,
	wl *gridtypes.WorkloadWithID,
) error {
	// - if Container, remount RW
	// prepare for container
	var (
		storage = stubs.NewStorageModuleStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
	)

	if err := flist.Unmount(ctx, wl.ID.String()); err != nil {
		return errors.Wrapf(err, "failed to unmount flist: %s", wl.ID.String())
	}
	rootfsSize := config.RootSize()
	// create a persisted volume for the vm. we don't do it automatically
	// via the flist, so we have control over when to decomission this volume.
	// remounting in RW mode
	volName := fmt.Sprintf("rootfs:%s", wl.ID.String())

	volumeExists, err := storage.VolumeExists(ctx, volName)
	if err != nil {
		return errors.Wrap(err, "failed to check if vm rootfs exists")
	}

	volume, err := storage.VolumeCreate(ctx, volName, rootfsSize)
	if err != nil {
		return errors.Wrap(err, "failed to create vm rootfs")
	}

	defer func() {
		if err != nil {
			// vm creation failed,
			if err := storage.VolumeDelete(ctx, volName); err != nil {
				log.Error().Err(err).Str("volume", volName).Msg("failed to delete persisted volume")
			}
		}
	}()

	mnt, err := flist.Mount(ctx, wl.ID.String(), config.FList, pkg.MountOptions{
		ReadOnly:        false,
		PersistedVolume: volume.Path,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to mount flist: %s", wl.ID.String())
	}

	// clean up host keys
	if !volumeExists {
		files, err := filepath.Glob(filepath.Join(mnt, "etc", "ssh", "ssh_host_*"))
		if err != nil {
			log.Debug().Err(err).Msg("failed to list ssh host keys for a vm image")
		}

		for _, file := range files {
			if err := os.Remove(file); err != nil {
				log.Debug().Err(err).Str("file", file).Msg("failed to delete host key file")
			}
		}
	}

	// inject container kernel and init
	machine.KernelImage = filepath.Join(cloudImage, "kernel")
	machine.InitrdImage = filepath.Join(cloudImage, "initramfs-linux.img")

	// can be overridden from the flist itself if exists
	if len(imageInfo.KernelPath) != 0 {
		machine.KernelImage = imageInfo.KernelPath
		machine.InitrdImage = imageInfo.InitrdPath
		// we are using kernel from flist, we need to respect
		// user init
		if len(config.Entrypoint) != 0 {
			machine.KernelArgs["init"] = config.Entrypoint
		}
	}

	machine.Boot = pkg.Boot{
		Type: pkg.BootVirtioFS,
		Path: mnt,
	}

	if err := fListStartup(config, filepath.Join(mnt, ".startup.toml")); err != nil {
		return errors.Wrap(err, "failed to apply startup config from flist")
	}

	if err := p.vmMounts(ctx, deployment, config.Mounts, true, machine); err != nil {
		return err
	}
	if config.Corex {
		if err := p.copyFile("/usr/bin/corex", filepath.Join(mnt, "corex"), 0755); err != nil {
			return errors.Wrap(err, "failed to inject corex binary")
		}
		machine.Entrypoint = "/corex --ipv6 -d 7 --interface eth0"
	}

	return nil
}

func (p *Manager) newMyceliumNetworkInterface(ctx context.Context, dl gridtypes.Deployment, wl *gridtypes.WorkloadWithID, config *zos.MyceliumIP) (pkg.VMIface, error) {
	network := stubs.NewNetworkerLightStub(p.zbus)
	netID := zos.NetworkID(dl.TwinID, config.Network)

	tapName := wl.ID.Unique(string(config.Network))
	iface, err := network.AttachMycelium(ctx, string(netID), tapName, config.Seed)

	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not set up tap device")
	}

	out := pkg.VMIface{
		Tap: iface.Name,
		MAC: iface.Mac.String(),
		IPs: []net.IPNet{
			*iface.IP,
		},
		Routes:     iface.Routes,
		PublicIPv4: false,
		PublicIPv6: false,
	}

	return out, nil
}

func (p *Manager) newPrivNetworkInterface(ctx context.Context, dl gridtypes.Deployment, wl *gridtypes.WorkloadWithID, inf zos.MachineInterface) (pkg.VMIface, error) {
	network := stubs.NewNetworkerLightStub(p.zbus)
	netID := zos.NetworkID(dl.TwinID, inf.Network)

	tapName := wl.ID.Unique(string(inf.Network))
	iface, err := network.AttachPrivate(ctx, string(netID), tapName, inf.IP)
	if err != nil {
		return pkg.VMIface{}, errors.Wrap(err, "could not set up tap device for private interface")
	}

	out := pkg.VMIface{
		Tap: iface.Name,
		MAC: iface.Mac.String(),
		IPs: []net.IPNet{
			*iface.IP,
			// privIP6,
		},
		Routes:            iface.Routes,
		IP4DefaultGateway: net.IP(iface.Routes[0].Gateway),
		// IP6DefaultGateway: gw6,
		PublicIPv4: false,
		PublicIPv6: false,
		NetID:      netID,
	}

	return out, nil
}

// FListInfo virtual machine details
type FListInfo struct {
	ImagePath  string
	KernelPath string
	InitrdPath string
}

func (t *FListInfo) IsContainer() bool {
	return len(t.ImagePath) == 0
}

func getFlistInfo(flistPath string) (flist FListInfo, err error) {
	files := map[string]*string{
		"/image.raw":       &flist.ImagePath,
		"/boot/vmlinuz":    &flist.KernelPath,
		"/boot/initrd.img": &flist.InitrdPath,
	}

	for rel, ptr := range files {
		path := filepath.Join(flistPath, rel)
		// this path can be a symlink so we need to make sure
		// the symlink is pointing to only files inside the
		// flist.

		// but we need to validate
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return flist, errors.Wrapf(err, "couldn't stat %s", rel)
		}

		if stat.IsDir() {
			return flist, fmt.Errorf("path '%s' cannot be a directory", rel)
		}
		mod := stat.Mode()
		switch mod.Type() {
		case 0:
			// regular file, do nothing
		case os.ModeSymlink:
			// this is a symlink. we
			// need to make sure it's a safe link
			// to a location inside the flist
			link, err := os.Readlink(path)
			if err != nil {
				return flist, errors.Wrapf(err, "failed to read link at '%s", rel)
			}
			// now the link if joined with path, (and cleaned) need to also point
			// to somewhere under flistPath
			abs := filepath.Clean(filepath.Join(flistPath, link))
			if !strings.HasPrefix(abs, flistPath) {
				return flist, fmt.Errorf("path '%s' points to invalid location", rel)
			}
		default:
			return flist, fmt.Errorf("path '%s' is of invalid type: %s", rel, mod.Type().String())
		}

		// set the value
		*ptr = path
	}

	return flist, nil
}

type startup struct {
	Entries map[string]entry `toml:"startup"`
}

type entry struct {
	Name string
	Args args
}

type args struct {
	Name string
	Dir  string
	Args []string
	Env  map[string]string
}

func (e entry) Entrypoint() string {
	if e.Name == "core.system" ||
		e.Name == "core.base" && e.Args.Name != "" {
		var buf strings.Builder

		buf.WriteString(e.Args.Name)
		for _, arg := range e.Args.Args {
			buf.WriteRune(' ')
			arg = strings.Replace(arg, "\"", "\\\"", -1)
			buf.WriteRune('"')
			buf.WriteString(arg)
			buf.WriteRune('"')
		}

		return buf.String()
	}

	return ""
}

func (e entry) WorkingDir() string {
	return e.Args.Dir
}

func (e entry) Envs() map[string]string {
	return e.Args.Env
}

// This code is backward compatible with flist .startup.toml file
// where the flist can define an Entrypoint and some initial environment
// variables. this is used *with* the container configuration like this
// - if no zmachine entry point is defined, use the one from .startup.toml
// - if envs are defined in flist, merge with the env variables from the
func fListStartup(data *zos.ZMachineLight, path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "failed to load startup file '%s'", path)
	}

	defer f.Close()

	log.Info().Msg("startup file found")
	startup := startup{}
	if _, err := toml.NewDecoder(f).Decode(&startup); err != nil {
		return err
	}

	entry, ok := startup.Entries["entry"]
	if !ok {
		return nil
	}

	data.Env = mergeEnvs(entry.Envs(), data.Env)

	if data.Entrypoint == "" && entry.Entrypoint() != "" {
		data.Entrypoint = entry.Entrypoint()
	}
	return nil
}

// mergeEnvs new into base
func mergeEnvs(base, new map[string]string) map[string]string {
	if len(base) == 0 {
		return new
	}

	for k, v := range new {
		base[k] = v
	}

	return base
}
