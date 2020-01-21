package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// Kubernetes reservation data
type Kubernetes struct {
	CPUCount uint8 `json:"cpu_count"`
	// Memory in KB for the vm
	Memory uint64 `json:"memory"`
	// DiskSize in MB
	DiskSize uint64 `json:"disk_size"`

	// NetworkID of the network namepsace in which to run the VM. The network
	// must be provisioned previously.
	NetworkID pkg.NetID `json:"network_id"`
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IP `json:"ip"`

	// ClusterSecret is the hex encoded encrypted cluster secret.
	ClusterSecret string `json:"cluster_secret"`
	// MasterIPs define the URL's for the kubernetes master nodes. If this
	// list is empty, this node is considered to be a master node.
	MasterIPs []net.IP `json:"master_ips"`
	// SSHKeys is a list of ssh keys to add to the VM. Keys can be either
	// a full ssh key, or in the form of `github:${username}`. In case of
	// the later, the VM will retrieve the github keys for this username
	// when it boots.
	SSHKeys []string `json:"ssh_keys"`

	PlainClusterSecret string `json:"-"`
}

// TODO: move flist
const k3osFlistURL = "https://hub.grid.tf/lee/k3os.flist"

func kubernetesProvision(ctx context.Context, reservation *Reservation) (interface{}, error) {
	return "", kubernetesProvisionImpl(ctx, reservation)
}

func kubernetesProvisionImpl(ctx context.Context, reservation *Reservation) error {
	var (
		client = GetZBus(ctx)

		storage = stubs.NewStorageModuleStub(client)
		network = stubs.NewNetworkerStub(client)
		flist   = stubs.NewFlisterStub(client)

		config Kubernetes
	)

	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	var err error
	config.PlainClusterSecret, err = decryptPassword(client, config.ClusterSecret)
	if err != nil {
		return errors.Wrap(err, "failed to decrypt namespace password")
	}

	// TODO: check if vm already exists

	var imagePath string
	imagePath, err = flist.Mount(k3osFlistURL, "", pkg.ReadOnlyMountOptions)
	if err != nil {
		return errors.Wrap(err, "could not mount k3os flist")
	}
	// In case of future errrors in the provisioning make sure we clean up
	defer func() {
		if err != nil {
			_ = flist.Umount(imagePath)
		}
	}()

	var storagePath string
	// disksize is in MB, make sure to convert
	storagePath, err = storage.CreateFilesystem(reservation.ID, config.DiskSize*1024*1024, pkg.SSDDevice)
	if err != nil {
		return errors.Wrap(err, "failed to reserve filesystem for vm")
	}
	defer func() {
		if err != nil {
			_ = storage.ReleaseFilesystem(storagePath)
		}
	}()

	var iface string
	iface, err = network.SetupTap(config.NetworkID)
	if err != nil {
		return errors.Wrap(err, "could not set up tap device")
	}
	defer func() {
		if err != nil {
			_ = network.RemoveTap(config.NetworkID, iface)
		}
	}()

	if err = kubernetesInstall(ctx, reservation.ID, storagePath, imagePath, iface, config); err != nil {
		return errors.Wrap(err, "failed to install k3s")
	}

	return kubernetesRun(ctx, reservation.ID, storagePath, imagePath, iface, config)

}

func kubernetesInstall(ctx context.Context, name string, storagePath string, imagePath string, iface string, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(GetZBus(ctx))

	cmdline := fmt.Sprintf("console=ttyS0 reboot=k panic=1 k3os.mode=install k3os.install.silent k3os.install.device=/dev/vda k3os.token=%s k3os.server_url=https://172.31.1.50:6443", cfg.PlainClusterSecret)
	// if there is no server url configured, the node is set up as a master, therefore
	// this will cause nodes with an empty master list to be implicitly treated as
	// a master node
	// TODO: check multi master setup
	for _, ip := range cfg.MasterIPs {
		var ipstring string
		if ip.To4() != nil {
			ipstring = ip.String()
		} else {
			ipstring = fmt.Sprintf("[%s]", ip.String())
		}
		cmdline = fmt.Sprintf("%s k3os.server_url=https://%s:6443", cmdline, ipstring)
	}
	for _, key := range cfg.SSHKeys {
		cmdline = fmt.Sprintf("%s ssh_authorized_keys=%s", cmdline, key)
	}

	installVM := pkg.VM{
		Name:   name,
		CPU:    cfg.CPUCount,
		Memory: int64(cfg.Memory),

		Storage:     storagePath,
		KernelImage: imagePath + "/k3os-vmlinux",
		KernelArgs:  cmdline,
		// TODO other args

		// TODO: Mount ISO
		Disks: []pkg.Disk{{Size: cfg.DiskSize}},
	}

	if err := vm.Run(installVM); err != nil {
		return err
	}

	// TODO: Currently the ISO does not support reboot from silent install, and
	// it will hang on shutdown. Install time is about 10 ish seconds on a decent system
	// so use a 1 min timeout for now to be sure, and manually kill the vm.
	time.Sleep(time.Minute * 1)

	// Delete the VM, the disk will be installed now
	return vm.Delete(name)
}

func kubernetesRun(ctx context.Context, name string, storagePath string, imagePath string, iface string, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(GetZBus(ctx))

	kubevm := pkg.VM{
		Name:   name,
		CPU:    cfg.CPUCount,
		Memory: int64(cfg.Memory),

		Storage:     storagePath,
		KernelImage: imagePath + "/k3os-vmlinux",
		KernelArgs:  "console=ttyS0 reboot=k panic=1",
		// TODO other args

		Disks: []pkg.Disk{{Size: cfg.DiskSize}},
	}

	return vm.Run(kubevm)
}
