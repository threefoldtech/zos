package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// KubernetesResult result returned by k3s reservation
type KubernetesResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// KubernetesCustomSize type
type KubernetesCustomSize struct {
	CRU int64   `json:"cru"`
	MRU float64 `json:"mru"`
	SRU float64 `json:"sru"`
}

// Kubernetes reservation data
type Kubernetes struct {
	VM `json:",inline"`

	// ClusterSecret is the hex encoded encrypted cluster secret.
	ClusterSecret string `json:"cluster_secret"`
	// MasterIPs define the URL's for the kubernetes master nodes. If this
	// list is empty, this node is considered to be a master node.
	MasterIPs []net.IP `json:"master_ips"`

	PlainClusterSecret string `json:"-"`

	DatastoreEndpoint     string `json:"datastore_endpoint"`
	DisableDefaultIngress bool   `json:"disable_default_ingress"`
}

// const k3osFlistURL = "https://hub.grid.tf/tf-official-apps/k3os.flist"
const k3osFlistURL = "https://hub.grid.tf/lee/k3os-ch.flist"

func (p *Provisioner) kubernetesProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.kubernetesProvisionImpl(ctx, reservation)
}

func ensureFList(flister pkg.Flister, url string) (string, error) {
	hash, err := flister.FlistHash(url)
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("k8s:%s", hash)

	return flister.NamedMount(name, url, "", pkg.ReadOnlyMountOptions)
}

func (p *Provisioner) kubernetesProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result KubernetesResult, err error) {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		network = stubs.NewNetworkerStub(p.zbus)
		flist   = stubs.NewFlisterStub(p.zbus)
		vm      = stubs.NewVMModuleStub(p.zbus)

		config Kubernetes

		needsInstall = true
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

	config.PlainClusterSecret, err = decryptSecret(config.ClusterSecret, reservation.User, reservation.Version, p.zbus)
	if err != nil {
		return result, errors.Wrap(err, "failed to decrypt namespace password")
	}
	if strings.ContainsAny(config.PlainClusterSecret, " \t\r\n\f") {
		return result, errors.New("cluster secret shouldn't contain whitespace chars")
	}
	cpu, memory, disk, err := vmSize(&config)
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	imagePath, err := ensureFList(flist, k3osFlistURL)
	if err != nil {
		return result, errors.Wrap(err, "could not mount k3os flist")
	}

	var diskPath string
	diskName := fmt.Sprintf("%s-%s", provision.FilesystemName(*reservation), "vda")
	if storage.Exists(diskName) {
		needsInstall = false
		info, err := storage.Inspect(diskName)
		if err != nil {
			return result, errors.Wrap(err, "could not get path to existing disk")
		}
		diskPath = info.Path
	} else {
		diskPath, err = storage.Allocate(diskName, int64(disk), "")
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
	netInfo, err = p.buildNetworkInfo(ctx, reservation.Version, reservation.User, iface, pubIface, config.VM)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	}

	if needsInstall {
		if err = p.kubernetesInstall(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config); err != nil {
			vm.Delete(reservation.ID)
			return result, errors.Wrap(err, "failed to install k3s")
		}
	}

	err = p.kubernetesRun(ctx, reservation.ID, cpu, memory, diskPath, imagePath, netInfo, config)
	if err != nil {
		// attempt to delete the vm, should the process still be lingering
		vm.Delete(reservation.ID)
	}

	return result, err
}

func (p *Provisioner) kubernetesInstall(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	cmdline := fmt.Sprintf("console=ttyS0 reboot=k panic=1 k3os.mode=install k3os.install.silent k3os.debug k3os.install.device=/dev/vda k3os.token=%s k3os.k3s_args=\"--flannel-iface=eth0\"", cfg.PlainClusterSecret)
	// if there is no server url configured, the node is set up as a master, therefore
	// this will cause nodes with an empty master list to be implicitly treated as
	// a master node
	for _, ip := range cfg.MasterIPs {
		var ipstring string
		if ip.To4() != nil {
			ipstring = ip.String()
		} else if ip.To16() != nil {
			ipstring = fmt.Sprintf("[%s]", ip.String())
		} else {
			return errors.New("invalid master IP")
		}
		cmdline = fmt.Sprintf("%s k3os.server_url=https://%s:6443", cmdline, ipstring)
	}
	for _, key := range cfg.SSHKeys {
		trimmed := strings.TrimSpace(key)
		if strings.ContainsAny(trimmed, "\"\n") {
			return errors.New("ssh keys shouldn't contain double quotes or intermediate new lines")
		}
		cmdline = fmt.Sprintf("%s ssh_authorized_keys=\"%s\"", cmdline, trimmed)
	}
	if cfg.DatastoreEndpoint != "" {
		cmdline = fmt.Sprintf("%s k3os.k3s_args=\"--datastore-endpoint=%s\"", cmdline, cfg.DatastoreEndpoint)
	}
	if cfg.DisableDefaultIngress {
		cmdline = fmt.Sprintf("%s k3os.k3s_args=\"--disable=traefik\"", cmdline)
	}

	disks := make([]pkg.VMDisk, 2)
	// install disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}
	// install ISO
	disks[1] = pkg.VMDisk{Path: imagePath + "/k3os-amd64.iso", ReadOnly: true, Root: false}
	installVM := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/k3os-vmlinux",
		InitrdImage: imagePath + "/k3os-initrd-amd64",
		KernelArgs:  cmdline,
		Disks:       disks,
		NoKeepAlive: true, //machine will not restarted automatically when it exists
	}

	if err := vm.Run(installVM); err != nil {
		return errors.Wrap(err, "could not run vm")
	}

	deadline, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()
	for {
		if !vm.Exists(name) {
			// install is done
			break
		}
		select {
		case <-time.After(time.Second * 3):
			// retry after 3 secs
		case <-deadline.Done():
			// If install takes longer than 5 minutes, we consider it a failure.
			// In that case, we attempt a delete first. This will kill the vm process
			// if it is still going. The actual resources (disk, taps, ...) should
			// be handled by the caller.
			logs, err := vm.Logs(name)
			if err != nil {
				log.Error().Err(err).Msg("failed to get machine logs")
			} else {
				log.Warn().Str("vm", name).Str("type", "machine-logs").Msg(logs)
			}

			if err := vm.Delete(name); err != nil {
				log.Warn().Err(err).Msg("could not delete vm who's install deadline expired")
			}
			return errors.New("failed to install vm in 5 minutes")
		}
	}

	// Delete the VM, the disk will be installed now
	return vm.Delete(name)
}

func (p *Provisioner) kubernetesRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Kubernetes) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	disks := make([]pkg.VMDisk, 1)
	// installed disk
	disks[0] = pkg.VMDisk{Path: diskPath, ReadOnly: false, Root: false}

	kubevm := pkg.VM{
		Name:        name,
		CPU:         cpu,
		Memory:      int64(memory),
		Network:     networkInfo,
		KernelImage: imagePath + "/k3os-vmlinux",
		InitrdImage: imagePath + "/k3os-initrd-amd64",
		KernelArgs:  "console=ttyS0 reboot=k panic=1",
		Disks:       disks,
	}

	return vm.Run(kubevm)
}

func (k *Kubernetes) Validate() error {
	err := k.VM.Validate()
	if err != nil {
		return err
	}
	for _, ip := range k.MasterIPs {
		if ip.To4() == nil && ip.To16() == nil {
			return errors.New("invalid master IP")
		}
	}
	return nil
}
