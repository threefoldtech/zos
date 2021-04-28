package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// KubernetesResult result returned by k3s reservation
type VMResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// Kubernetes reservation data
type VM struct {
	// Size of the vm, this defines the amount of vCpu, memory, and the disk size
	// Docs: docs/kubernetes/sizes.md
	Size int64 `json:"size"`

	Custom VMCustomSize `json:"custom_size"`
	// NetworkID of the network namepsace in which to run the VM. The network
	// must be provisioned previously.
	NetworkID pkg.NetID `json:"network_id"`
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IP `json:"ip"`

	SSHKeys []string `json:"ssh_keys"`
	// PublicIP points to a reservation for a public ip
	PublicIP schema.ID `json:"public_ip"`

	// A name of a predefined list of VMs
	Name string `json:"name"`
}

type VMInfo struct {
	Flist              string
	Initrd             string
	Kernel             string
	CmdlineConstructor func(VM) string
}

// const k3osFlistURL = "https://hub.grid.tf/tf-official-apps/k3os.flist"
var flistMap = map[string]VMInfo{
	"ubuntu-20": VMInfo{
		Flist:              "https://hub.grid.tf/omar0.3bot/omarelawady-zos-ubuntu-vm-latest.flist",
		Initrd:             "initrd.img-5.8.0-34-generic",
		Kernel:             "vmlinuz-5.8.0-34-generic",
		CmdlineConstructor: ubuntuCMDLine,
	},
}

func (p *Provisioner) VirtualMachineProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.virtualMachineProvisionImpl(ctx, reservation)
}

func (p *Provisioner) virtualMachineProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result KubernetesResult, err error) {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config VM
	)
	log.Debug().Str("data", string(reservation.Data)).Msg("Received Data")
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	netID := provision.NetworkID(reservation.User, string(config.NetworkID))

	// check if the network tap already exists
	// if it does, it's most likely that a vm with the same network id and node id already exists
	// this will cause the reservation to fail
	exists, err := network.TapExists(netID)
	if err != nil {
		return result, errors.Wrap(err, "could not check if tap device exists")
	}

	if exists {
		return result, errors.New("kubernetes vm with same network already exists")
	}

	// check if public ipv4 is supported, should this be requested
	if config.PublicIP != 0 && !network.PublicIPv4Support() {
		return result, errors.New("public ipv4 is requested, but not supported on this node")
	}

	result.ID = reservation.ID
	result.IP = config.IP.String()

	cpu, memory, disk, err := vmSize(&config)
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}
	imageInfo := flistMap[config.Name]
	imagePath, err := ensureFList(flist, imageInfo.Flist)
	if err != nil {
		return result, errors.Wrap(err, "could not mount k3os flist")
	}

	var diskPath string
	diskName := fmt.Sprintf("%s-%s", provision.FilesystemName(*reservation), "vda")
	if storage.Exists(diskName) {
		info, err := storage.Inspect(diskName)
		if err != nil {
			return result, errors.Wrap(err, "could not get path to existing disk")
		}
		diskPath = info.Path
	} else {
		diskPath, err = storage.Allocate(diskName, int64(disk))
		if err != nil {
			return result, errors.Wrap(err, "failed to reserve filesystem for vm")
		}
	}
	// clean up the disk anyway, even if it has already been installed.
	defer func() {
		if err != nil {
			_ = storage.Deallocate(diskName)
		}
	}()

	var iface string
	iface, err = network.SetupTap(netID)
	if err != nil {
		return result, errors.Wrap(err, "could not set up tap device")
	}

	defer func() {
		if err != nil {
			_ = network.RemoveTap(netID)
		}
	}()

	var pubIface string
	if config.PublicIP != 0 {
		pubIface, err = network.SetupPubTap(pubIPResID(config.PublicIP))
		if err != nil {
			return result, errors.Wrap(err, "could not set up tap device for public network")
		}

		defer func() {
			if err != nil {
				_ = network.RemovePubTap(pubIPResID(config.PublicIP))
			}
		}()
	}

	var netInfo pkg.VMNetworkInfo
	netInfo, err = p.buildNetworkInfo(ctx, reservation.Version, reservation.User, iface, pubIface, config.IP, config.PublicIP, config.NetworkID)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}
	err = p.prepareVMFS(ctx, imagePath, diskPath)
	if err != nil {
		return result, err
	}
	cmdline := imageInfo.CmdlineConstructor(config)
	err = p.vmRun(ctx, reservation.ID, cpu, memory, diskPath, imagePath, imageInfo.Initrd, imageInfo.Kernel, cmdline, netInfo)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		vm.Delete(reservation.ID)
	}

	return result, err
}

func (p *Provisioner) prepareVMFS(ctx context.Context, imagePath string, diskPath string) error {

	var cmd *exec.Cmd
	imageFlag := fmt.Sprintf("if=%s", imagePath+"/ubuntu.raw")
	diskFlag := fmt.Sprintf("of=%s", diskPath)
	cmd = exec.CommandContext(ctx, "dd", imageFlag, diskFlag, "conv=notrunc")

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to copy the ubuntu raw image over the disk")
	}
	dname, err := ioutil.TempDir("", "btrfs-resize")
	if err != nil {
		return errors.Wrap(err, "couldn't create a temp dir to mount the btrfs fs to resize it")
	}
	defer os.RemoveAll(dname)

	cmd = exec.CommandContext(ctx, "mount", diskPath, dname)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "couldn't mount the created disk to a tmp dir")
	}

	defer func() {
		cmd = exec.CommandContext(ctx, "umount", dname)

		if err := cmd.Run(); err != nil {
			log.Error().Str("path", dname).Msg("Couldn't umount the tmp btrfs")
		}
	}()

	cmd = exec.CommandContext(ctx, "btrfs", "filesystem", "resize", "max", dname)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to resize file system to disk size")
	}
	return nil
}

func (p *Provisioner) vmRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, initrd string, kernel string, cmdline string, networkInfo pkg.VMNetworkInfo) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	disks := make([]pkg.VMDisk, 1)
	// installed disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}
	kubevm := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/" + initrd,
		InitrdImage: imagePath + "/" + kernel,
		KernelArgs:  cmdline,
		Disks:       disks,
	}

	return vm.Run(kubevm)
}

func (k *VM) GetSize() int64 {
	return k.Size
}

func (k *VM) GetCustomSize() VMCustomSize {
	return k.Custom
}

func ubuntuCMDLine(config VM) string {
	sshKey := ""
	if len(config.SSHKeys) > 0 {
		sshKey = config.SSHKeys[0]
	}
	cmdline := "root=/dev/vda rw console=ttyS0 reboot=k panic=1"
	cmdline = fmt.Sprintf("%s ssh=%s", cmdline, strings.Replace(sshKey, " ", ",", -1))
	return cmdline
}
