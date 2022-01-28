package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfexplorer/schema"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// VMResult result returned by k3s reservation
type VMResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// VM reservation data
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

// VMInfo kernel initrd and the raw disk path of the vm
type VMInfo struct {
	Initrd    string
	Kernel    string
	ImagePath string
}

// VMREPO in which all the vm flists are stored
const VMREPO = "https://hub.grid.tf/tf-official-vms/"

// VMTAG the tag of the vm images
const VMTAG = "latest"

func (p *Provisioner) virtualMachineProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
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
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	if err = config.Validate(); err != nil {
		return result, err
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
		return result, errors.New("another vm with same network already exists")
	}

	// check if public ipv4 is supported, should this be requested
	if config.PublicIP != 0 && !network.PublicIPv4Support() {
		return result, errors.New("public ipv4 is requested, but not supported on this node")
	}

	result.ID = reservation.ID
	result.IP = config.IP.String()

	cpu, memory, disk, err := vmSize(config)
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	flistName := VMREPO + strings.ToLower(config.Name) + "-" + VMTAG + ".flist"
	imagePath, err := ensureFList(flist, flistName)
	if err != nil {
		return result, errors.Wrap(err, "could not mount k3os flist")
	}
	imageInfo, err := constructImageInfo(imagePath)
	if err != nil {
		return result, err
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
		diskPath, err = storage.Allocate(diskName, int64(disk), imageInfo.ImagePath)
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
	netInfo, err = p.buildNetworkInfo(ctx, reservation.Version, reservation.User, iface, pubIface, config)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}
	cmdline, err := constructCMDLine(config)
	if err != nil {
		return result, err
	}
	err = p.vmRun(ctx, reservation.ID, cpu, memory, diskPath, imageInfo, cmdline, netInfo)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		_ = vm.Delete(reservation.ID)
	}

	return result, err
}

func (p *Provisioner) vmRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imageInfo VMInfo, cmdline string, networkInfo pkg.VMNetworkInfo) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	disks := make([]pkg.VMDisk, 1)
	// installed disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}
	vmObj := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imageInfo.Kernel,
		InitrdImage: imageInfo.Initrd,
		KernelArgs:  cmdline,
		Disks:       disks,
	}

	return vm.Run(vmObj)
}

func constructCMDLine(config VM) (string, error) {
	cmdline := "root=/dev/vda rw console=ttyS0 reboot=k panic=1"
	for _, key := range config.SSHKeys {
		trimmed := strings.TrimSpace(key)
		if strings.ContainsAny(trimmed, "\t\r\n\f") {
			return "", errors.New("ssh keys can't contain intermediate whitespace chars other than white space")
		}
		cmdline = fmt.Sprintf("%s ssh=%s", cmdline, strings.Replace(trimmed, " ", ",", -1))
	}
	return cmdline, nil
}

func constructImageInfo(imagePath string) (VMInfo, error) {
	initrd := ""
	if _, err := os.Stat(imagePath + "/initrd"); err == nil {
		initrd = imagePath + "/initrd"
	}
	kernel := imagePath + "/kernel"
	if _, err := os.Stat(kernel); err != nil {
		return VMInfo{}, errors.Wrap(err, "couldn't stat kernel")
	}
	image := imagePath + "/image.raw"
	if _, err := os.Stat(image); err != nil {
		return VMInfo{}, errors.Wrap(err, "couldn't stat image.raw")
	}
	return VMInfo{Initrd: initrd, Kernel: kernel, ImagePath: image}, nil
}

// Validate validates the vm config, name, ip, size, and ssh keys
func (k *VM) Validate() error {
	// avoid funny stuff like ../badaccount/another-flist
	if matched, _ := regexp.MatchString("^[0-9a-zA-Z-.]*$", k.Name); !matched {
		return errors.New("the name must consist of alphanumeric characters, dot, and dash ony")
	}

	if k.IP.To4() == nil && k.IP.To16() == nil {
		return errors.New("invalid IP")
	}
	if k.Size != -1 && (k.Size < 1 || k.Size > 18) {
		return errors.New("unsupported vm size %d, only size -1, and 1 to 18 are supported")
	}
	for _, key := range k.SSHKeys {
		trimmed := strings.TrimSpace(key)
		if strings.ContainsAny(trimmed, "\t\r\n\f\"") {
			return errors.New("ssh keys can't contain intermediate whitespace chars other than white space")
		}
	}
	return nil
}
