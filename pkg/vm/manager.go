package vm

import (
	"context"
	"fmt"
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
	"os"
	"path/filepath"
	"strings"
)

const (
	// FCBin  path for firecracker
	FCBin             = "/bin/firecracker"
	FCSockDir         = "/var/run/firecracker"
	defaultKernelArgs = "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules"
)

// vmModuleImpl implements the VMModule interface
type vmModuleImpl struct{}

var (
	_ pkg.VMModule = (*vmModuleImpl)(nil)
)

// NewVMModule creates a new instance of vm manager
func NewVMModule() (pkg.VMModule, error) {
	if err := os.MkdirAll(FCSockDir, 0755); err != nil {
		return nil, err
	}

	return &vmModuleImpl{}, nil
}

func (m *vmModuleImpl) makeDisk(name string, size int64) error {
	disk, err := os.Create(name)
	if err != nil {
		return errors.Wrapf(err, "failed to create disk '%s'", name)
	}
	defer disk.Close()
	if err := disk.Truncate(size * 1024 * 1024); err != nil {
		return errors.Wrapf(err, "failed to truncate disk file '%s'", name)
	}

	return nil
}

func (m *vmModuleImpl) makeDevices(vm *pkg.VM) ([]models.Drive, error) {
	drives := []models.Drive{
		{
			DriveID:      firecracker.String("1"),
			IsReadOnly:   firecracker.Bool(false),
			IsRootDevice: firecracker.Bool(true),
			PathOnHost:   firecracker.String(vm.RootImage),
		},
	}

	for i, disk := range vm.Disks {
		id := fmt.Sprintf("%d", i+2)
		path := filepath.Join(vm.Storage, fmt.Sprintf("%s.disk", id))
		if err := m.makeDisk(path, int64(disk.Size)); err != nil {
			return nil, err
		}

		drives = append(drives, models.Drive{
			DriveID:      firecracker.String(id),
			IsReadOnly:   firecracker.Bool(false),
			IsRootDevice: firecracker.Bool(false),
			PathOnHost:   firecracker.String(path),
		})
	}

	return drives, nil
}

func (m *vmModuleImpl) skipName(n string) string {
	return strings.ReplaceAll(n, string(filepath.Separator), "-")
}

// Run vm
func (m *vmModuleImpl) Run(vm pkg.VM) error {
	if err := vm.Validate(); err != nil {
		return errors.Wrap(err, "machine configuration validation failed")
	}

	ctx := context.Background()

	socket := filepath.Join(FCSockDir, fmt.Sprintf("fc.%s.sock", m.skipName(vm.Name)))
	if _, err := os.Stat(socket); err == nil {
		return fmt.Errorf("a vm with same name already exists")
	}

	devices, err := m.makeDevices(&vm)
	if err != nil {
		return err
	}

	if len(vm.KernelArgs) == 0 {
		vm.KernelArgs = defaultKernelArgs
	}

	cfg := firecracker.Config{
		SocketPath:      socket,
		KernelImagePath: vm.KernelImage,
		KernelArgs:      vm.KernelArgs,
		MachineCfg: models.MachineConfiguration{
			HtEnabled:  firecracker.Bool(false),
			MemSizeMib: firecracker.Int64(vm.Memory),
			VcpuCount:  firecracker.Int64(int64(vm.CPU)),
		},
		Drives: devices,
	}

	log.Debug().Msgf("Machine Config: %+v", cfg)

	out, err := os.Create(filepath.Join(vm.Storage, "machine.log"))
	if err != nil {
		return err
	}

	defer out.Close()

	cmd := firecracker.VMCommandBuilder{}.
		WithBin(FCBin).
		WithSocketPath(socket).
		WithStdout(out).
		WithStderr(out).
		Build(ctx)

	var opts []firecracker.Opt
	opts = append(opts, firecracker.WithProcessRunner(cmd))

	machine, err := firecracker.NewMachine(ctx, cfg, opts...)
	if err != nil {
		return err
	}

	return machine.Start(ctx)
}

func (m *vmModuleImpl) Inspect(name string) (pkg.VM, error) {
	return pkg.VM{}, nil
}

func (m *vmModuleImpl) Delete(name string) error {
	return nil
}
