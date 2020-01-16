package vm

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg"
)

const (
	// FCBin  path for firecracker
	FCBin = "/bin/firecracker"
	// JailerBin path for fc jailer
	JailerBin = "/bin/jailer"
	// FCSockDir where vm firecracker sockets are kept
	FCSockDir = "/var/run/firecracker"

	defaultKernelArgs = "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules"
)

// vmModuleImpl implements the VMModule interface
type vmModuleImpl struct {
	root string
}

var (
	_ pkg.VMModule = (*vmModuleImpl)(nil)
)

// NewVMModule creates a new instance of vm manager
func NewVMModule(root string) (pkg.VMModule, error) {
	if err := os.MkdirAll(FCSockDir, 0755); err != nil {
		return nil, err
	}

	return &vmModuleImpl{
		root: root,
	}, nil
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

func (m *vmModuleImpl) machineRoot(id string) string {
	return filepath.Join(m.root, "firecracker", id, "root")
}

func (m *vmModuleImpl) socket(id string) string {
	return filepath.Join(m.machineRoot(id), "api.socket")
}

func (m *vmModuleImpl) exists(id string) bool {
	socket := m.socket(id)
	con, err := net.Dial("unix", socket)
	if err != nil {
		return false
	}

	con.Close()
	return true
}

func (m *vmModuleImpl) cleanFs(id string) error {
	root := m.machineRoot(id)
	stat, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("invalid machine root, expecting a directory")
	}

	// do all un-mounting
	return os.RemoveAll(root)
}

// Run vm
func (m *vmModuleImpl) Run(vm pkg.VM) error {
	if err := vm.Validate(); err != nil {
		return errors.Wrap(err, "machine configuration validation failed")
	}

	ctx := context.Background()

	if m.exists(vm.Name) {
		return fmt.Errorf("a vm with same name already exists")
	}

	// make sure to clean up previous roots just in case
	if err := m.cleanFs(vm.Name); err != nil {
		return err
	}

	devices, err := m.makeDevices(&vm)
	if err != nil {
		return err
	}

	if len(vm.KernelArgs) == 0 {
		vm.KernelArgs = defaultKernelArgs
	}

	out, err := os.Create(filepath.Join(vm.Storage, "machine.log"))
	if err != nil {
		return err
	}

	cfg := firecracker.Config{
		KernelImagePath: vm.KernelImage,
		KernelArgs:      vm.KernelArgs,
		MachineCfg: models.MachineConfiguration{
			HtEnabled:  firecracker.Bool(false),
			MemSizeMib: firecracker.Int64(vm.Memory),
			VcpuCount:  firecracker.Int64(int64(vm.CPU)),
		},
		JailerCfg: &firecracker.JailerConfig{
			UID:           firecracker.Int(0),
			GID:           firecracker.Int(0),
			NumaNode:      firecracker.Int(0),
			ExecFile:      FCBin,
			JailerBinary:  JailerBin,
			ID:            vm.Name,
			ChrootBaseDir: m.root,
			Daemonize:     true,
			// we probably need to change that to use mount instead
			// of hard link
			ChrootStrategy: firecracker.NewNaiveChrootStrategy(
				filepath.Join(m.root, "firecracker", vm.Name),
				"vmlinuz",
			),
			Stdout: out,
			Stderr: out,
		},
		Drives:         devices,
		ForwardSignals: []os.Signal{}, // it has to be an empty list to prevent using the default
	}

	log.Debug().Msgf("Machine Config: %+v", cfg)

	cmd := firecracker.JailerCommandBuilder{}.
		WithBin(JailerBin).
		WithChrootBaseDir(m.root).
		WithDaemonize(true).
		WithID(vm.Name).
		WithStdout(out).
		WithStderr(out).
		WithExecFile(FCBin).
		Build(ctx)

	defer out.Close()

	var opts []firecracker.Opt
	opts = append(opts, firecracker.WithProcessRunner(cmd))

	machine, err := firecracker.NewMachine(ctx, cfg, opts...)

	if err != nil {
		return err
	}

	return machine.Start(ctx)
}

func (m *vmModuleImpl) Inspect(name string) (pkg.VMInfo, error) {
	if !m.exists(name) {
		return pkg.VMInfo{}, fmt.Errorf("machine '%s' does not exist", name)
	}

	client := firecracker.NewClient(m.socket(name), nil, false)

	cfg, err := client.GetMachineConfiguration()
	if err != nil {
		return pkg.VMInfo{}, errors.Wrap(err, "failed to get machine configuration")
	}

	return pkg.VMInfo{
		CPU:       *cfg.Payload.VcpuCount,
		Memory:    *cfg.Payload.MemSizeMib,
		HtEnabled: *cfg.Payload.HtEnabled,
	}, nil
}

func (m *vmModuleImpl) Delete(name string) error {
	return nil
}
