package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
)

// QemuResult result returned by qemu reservation
type QemuResult struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// Qemu reservation data
type Qemu struct {
	// NetworkID of the network namepsace in which to run the VM. The network
	// must be provisioned previously.
	NetworkID pkg.NetID `json:"network_id"`
	// IP of the VM. The IP must be part of the subnet available in the network
	// resource defined by the networkID on this node
	IP net.IP `json:"ip"`
	// Image of the VM.
	Image string `json:"image"`
	// QemuCapacity is the amount of resource to allocate to the virtual machine
	Capacity QemuCapacity `json:"capacity"`
}

// QemuCapacity is the amount of resource to allocate to the virtual machine
type QemuCapacity struct {
	// Number of CPU
	CPU uint8 `json:"cpu"`
	// Memory in MiB
	Memory uint64 `json:"memory"`
	// HDD in GB
	HDDSize uint64 `json:"hdd"`
}

const qemuFlistURL = "https://hub.grid.tf/maximevanhees.3bot/qemu-flist-tarball.flist"

func (p *Provisioner) qemuProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	return p.qemuProvisionImpl(ctx, reservation)
}

func (p *Provisioner) qemuProvisionImpl(ctx context.Context, reservation *provision.Reservation) (result QemuResult, err error) {
	var (
		storage = stubs.NewVDiskModuleStub(p.zbus)
		//network = stubs.NewNetworkerStub(p.zbus)
		flist = stubs.NewFlisterStub(p.zbus)
		vm    = stubs.NewVMModuleStub(p.zbus)

		config Qemu

		needsInstall = true
	)

	// checking reservation scheme and putting data in config
	if err := json.Unmarshal(reservation.Data, &config); err != nil {
		return result, errors.Wrap(err, "failed to decode reservation schema")
	}

	// mounting image
	var mnt string
	mnt, err = flist.Mount(config.Image, "", pkg.DefaultMountOptions)
	if err != nil {
		return QemuResult{}, err
	}
	// unmount flist if error occurs
	defer func() {
		if err != nil {
			if err := flist.Umount(mnt); err != nil {
				log.Error().Err(err).Str("mnt", mnt).Msg("failed to unmount")
			}
		}
	}()

	// set IP address
	result.IP = config.IP.String()
	// set ID
	result.ID = reservation.ID

	cpu, memory, disk, err := qemuCapacitySize(config.Capacity)
	if err != nil {
		return result, errors.Wrap(err, "could not interpret vm size")
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	if _, err = vm.Inspect(reservation.ID); err == nil {
		// vm is already running, nothing to do here
		return result, nil
	}

	var imagePath string
	imagePath, err = flist.NamedMount(reservation.ID, qemuFlistURL, "", pkg.ReadOnlyMountOptions)
	if err != nil {
		return result, errors.Wrap(err, "could not mount qemu flist")
	}
	// In case of future errrors in the provisioning make sure we clean up
	defer func() {
		if err != nil {
			_ = flist.Umount(imagePath)
		}
	}()

	var diskPath string
	diskName := fmt.Sprintf("%s-%s", reservation.ID, "vda")
	if storage.Exists(diskName) {
		needsInstall = false
		info, err := storage.Inspect(diskName)
		if err != nil {
			return result, errors.Wrap(err, "could not get path to existing disk")
		}
		diskName = info.Path
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

	// NETWORK THINGS HAVE TO BE DONE
	// ___________________________________________________________________________________________
	/* // setup tap device
	var iface string
	netID := networkID(reservation.User, string(config.NetworkID))
	iface, err = network.SetupTap(netID)
	if err != nil {
		return result, errors.Wrap(err, "could not set up tap device")
	}

	defer func() {
		if err != nil {
			_ = vm.Delete(reservation.ID)
			_ = network.RemoveTap(netID)
		}
	}()

	var netInfo pkg.VMNetworkInfo
	netInfo, err = p.buildNetworkInfo(ctx, reservation.User, iface, config)
	if err != nil {
		return result, errors.Wrap(err, "could not generate network info")
	} */
	// ___________________________________________________________________________________________

	if needsInstall {
		if err = p.qemuInstall(ctx, reservation.ID, cpu, memory, diskPath, imagePath, config); err != nil {
			return result, errors.Wrap(err, "failed to install qemu")
		}
	}

	err = p.qemuRun(ctx, reservation.ID, cpu, memory, diskPath, imagePath, config)

	return result, err
}

// with network info
//func (p *Provisioner) qemuInstall(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Qemu) error {

//without network info
func (p *Provisioner) qemuInstall(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, cfg Qemu) error {
	// prepare disks here
	// ....

	vm := stubs.NewVMModuleStub(p.zbus)

	//cmdline = fmt.Sprintf()

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
			return errors.New("failed to install vm in 5 minutes")
		}
	}

	// Delete the VM, the disk will be installed now
	return vm.Delete(name)
	//return nil
}

// with network info
//func (p *Provisioner) qemuRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, networkInfo pkg.VMNetworkInfo, cfg Qemu) error {
// witouth network info
func (p *Provisioner) qemuRun(ctx context.Context, name string, cpu uint8, memory uint64, diskPath string, imagePath string, cfg Qemu) error {
	vm := stubs.NewVMModuleStub(p.zbus)

	// create virtual machine and run it
	qemuVM := pkg.QemuVM{}

	return vm.Run(&qemuVM)
	//return nil
}

func (p *Provisioner) qemuDecommision(ctx context.Context, reservation *provision.Reservation) error {
	return nil
}

func qemuCapacitySize(qc QemuCapacity) (uint8, uint64, uint64, error) {
	return qc.CPU, qc.Memory, qc.HDDSize, nil
}
