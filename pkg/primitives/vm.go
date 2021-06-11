package primitives

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jbenet/go-base58"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// VirtualMachine type
type VirtualMachine = zos.VirtualMachine

// VirtualMachineResult type
type VirtualMachineResult = zos.VirtualMachineResult

// VMInfo virtual machine details
type VMInfo struct {
	Initrd    string
	Kernel    string
	ImagePath string
}

// VMREPO official Threefold virtual machines repo
const VMREPO = "https://hub.grid.tf/tf-official-vms/"

// VMTAG tag for all VMs on this repo
const VMTAG = "latest"

func (p *Primitives) virtualMachineProvision(ctx context.Context, wl *gridtypes.WorkloadWithID) (interface{}, error) {
	return p.virtualMachineProvisionImpl(ctx, wl)
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

	cap, err := config.Capacity()
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(ctx, wl.ID.String()); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	flistName := VMREPO + strings.ToLower(config.Name) + "-" + VMTAG + ".flist"
	imagePath, err := ensureFList(ctx, flist, flistName)
	if err != nil {
		return result, errors.Wrap(err, "could not mount vm flist")
	}
	imageInfo, err := constructImageInfo(imagePath)
	if err != nil {
		return result, err
	}

	var diskPath string
	diskName := fmt.Sprintf("%s-%s", FilesystemName(wl), "vda")
	if storage.Exists(ctx, diskName) {
		info, err := storage.Inspect(ctx, diskName)
		if err != nil {
			return result, errors.Wrap(err, "could not get path to existing disk")
		}
		diskPath = info.Path
	} else {
		diskPath, err = storage.Allocate(ctx, diskName, cap.SRU, imageInfo.ImagePath)
		if err != nil {
			return result, errors.Wrap(err, "failed to reserve filesystem for vm")
		}
	}
	// clean up the disk anyway, even if it has already been installed.
	defer func() {
		if err != nil {
			_ = storage.Deallocate(ctx, diskName)
		}
	}()

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
	cmdline, err := constructCMDLine(config)
	if err != nil {
		return result, err
	}
	err = p.vmRun(ctx, wl.ID.String(), uint8(cap.CRU), cap.MRU, diskPath, imageInfo, cmdline, netInfo)
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

func (p *Primitives) vmRun(ctx context.Context, name string, cpu uint8, memory gridtypes.Unit, diskPath string, imageInfo VMInfo, cmdline string, networkInfo pkg.VMNetworkInfo) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	disks := make([]pkg.VMDisk, 1)
	// installed disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}
	kubevm := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      memory,
		Network:     networkInfo,
		KernelImage: imageInfo.Kernel,
		InitrdImage: imageInfo.Initrd,
		KernelArgs:  cmdline,
		Disks:       disks,
	}

	return vm.Run(ctx, kubevm)
}

func constructCMDLine(config VirtualMachine) (string, error) {
	cmdline := "root=/dev/vda rw console=ttyS0 console=hvc0 reboot=k panic=1"
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
