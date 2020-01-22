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
	// Size of the vm, this defines the amount of vCpu, memory, and the disk size
	Size uint8 `json:"size"`

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
	return nil, kubernetesProvisionImpl(ctx, reservation)
}

func kubernetesProvisionImpl(ctx context.Context, reservation *Reservation) error {
	var (
		client = GetZBus(ctx)

		storage = stubs.NewStorageModuleStub(client)
		network = stubs.NewNetworkerStub(client)
		flist   = stubs.NewFlisterStub(client)
		vm      = stubs.NewVMModuleStub(client)

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

	cpu, memory, disk, err := vmSize(config.Size)
	if err != nil {
		return errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return nil
	}

	var imagePath string
	imagePath, err = flist.NamedMount(reservation.ID, k3osFlistURL, "", pkg.ReadOnlyMountOptions)
	if err != nil {
		return errors.Wrap(err, "could not mount k3os flist")
	}
	// In case of future errrors in the provisioning make sure we clean up
	defer func() {
		if err != nil {
			_ = flist.Umount(imagePath)
		}
	}()

	// TODO: replace with vDisk alloc
	var storagePath string
	// disksize is in MB, make sure to convert
	storagePath, err = storage.CreateFilesystem(reservation.ID, disk*1024*1024, pkg.SSDDevice)
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
			_ = network.RemoveTap(config.NetworkID)
		}
	}()

	var netInfo pkg.VMNetworkInfo
	netInfo, err = buildNetworkInfo(ctx, iface, config)
	if err != nil {
		return errors.Wrap(err, "could not generate network info")
	}

	if err = kubernetesInstall(ctx, reservation.ID, cpu, memory, storagePath, imagePath, netInfo, config); err != nil {
		return errors.Wrap(err, "failed to install k3s")
	}

	return kubernetesRun(ctx, reservation.ID, cpu, memory, storagePath, imagePath, netInfo, config)
}

func kubernetesInstall(ctx context.Context, name string, cpu uint8, memory uint64, storagePath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(GetZBus(ctx))

	cmdline := fmt.Sprintf("console=ttyS0 reboot=k panic=1 k3os.mode=install k3os.install.silent k3os.install.device=/dev/vda k3os.token=%s k3os.server_url=https://172.31.1.50:6443", cfg.PlainClusterSecret)
	// if there is no server url configured, the node is set up as a master, therefore
	// this will cause nodes with an empty master list to be implicitly treated as
	// a master node
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
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/k3os-vmlinux",
		InitrdImage: imagePath + "/k3os-initrd-amd64",
		KernelArgs:  cmdline,
		// TODO: Mount ISO
		// Disks: []pkg.Disk{{Size: cfg.DiskSize}},
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

func kubernetesRun(ctx context.Context, name string, cpu uint8, memory uint64, storagePath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(GetZBus(ctx))

	kubevm := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/k3os-vmlinux",
		InitrdImage: imagePath + "/k3os-initrd-amd64",
		KernelArgs:  "console=ttyS0 reboot=k panic=1",
		// TODO disks
		// Disks: []pkg.Disk{{Size: cfg.DiskSize}},
	}

	return vm.Run(kubevm)
}

func kubernetesDecomission(ctx context.Context, reservation *Reservation) error {
	var (
		client = GetZBus(ctx)

		// storage = stubs.NewStorageModuleStub(client)
		network = stubs.NewNetworkerStub(client)
		flist   = stubs.NewFlisterStub(client)
		vm      = stubs.NewVMModuleStub(client)

		cfg Kubernetes
	)

	if err := json.Unmarshal(reservation.Data, &cfg); err != nil {
		return errors.Wrap(err, "failed to decode reservation schema")
	}

	if _, err := vm.Inspect(reservation.ID); err == nil {
		if err := vm.Delete(reservation.ID); err != nil {
			return errors.Wrapf(err, "failed to delete vm %s", reservation.ID)
		}
	}

	if err := network.RemoveTap(cfg.NetworkID); err != nil {
		return errors.Wrap(err, "could not clean up tap device")
	}

	// TODO clean up storage

	if err := flist.NamedUmount(reservation.ID); err != nil {
		return errors.Wrap(err, "could not unmount flist")
	}

	return nil
}

func buildNetworkInfo(ctx context.Context, iface string, cfg Kubernetes) (pkg.VMNetworkInfo, error) {
	network := stubs.NewNetworkerStub(GetZBus(ctx))

	subnet, err := network.GetSubnet(cfg.NetworkID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network resource subnet")
	}

	if !subnet.Contains(cfg.IP) {
		return pkg.VMNetworkInfo{}, fmt.Errorf("IP %s is not part of local nr subnet %s", cfg.IP.String(), subnet.String())
	}

	addrCIDR := net.IPNet{
		IP:   cfg.IP,
		Mask: subnet.Mask,
	}

	gw, err := network.GetDefaultGwIP(cfg.NetworkID)
	if err != nil {
		return pkg.VMNetworkInfo{}, errors.Wrapf(err, "could not get network resource default gateway")
	}

	networkInfo := pkg.VMNetworkInfo{
		Tap:         iface,
		MAC:         "", // rely on static IP configuration so we don't care here
		AddressCIDR: addrCIDR.String(),
		GatewayIP:   net.IP(gw).String(),
		Nameservers: []string{"8.8.8.8", "8.8.4.4"},
	}

	return networkInfo, nil
}

// returns the vCpu's, memory, disksize for a vm size
// memory and disk size is expressed in MiB
func vmSize(size uint8) (uint8, uint64, uint64, error) {
	switch size {
	case 1:
		// 1 vCpu, 2 GiB RAM, 50 GB disk
		return 1, 2 * 1024, 50 * 1024, nil
	case 2:
		// 2 vCpu, 4 GiB RAM, 100 GB disk
		return 2, 4 * 1024, 100 * 1024, nil
	}

	return 0, 0, 0, fmt.Errorf("unsupported vm size %d, only size 1 and 2 are supported", size)
}
