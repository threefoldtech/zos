package vm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

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

	errScanFound = fmt.Errorf("found")
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
	return filepath.Join(m.root, "firecracker", id)
}

func (m *vmModuleImpl) socket(id string) string {
	return filepath.Join(m.machineRoot(id), "root", "api.socket")
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
	root := filepath.Join(m.machineRoot(id), "root")

	files, err := ioutil.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	for _, entry := range files {
		if entry.IsDir() {
			continue
		}

		// we try to unmount every file in the directory
		// because it's faster than trying to find exactly
		// what files are mounted under this location.
		path := filepath.Join(root, entry.Name())
		err := syscall.Unmount(
			path,
			syscall.MNT_DETACH,
		)

		if err != nil {
			log.Warn().Err(err).Str("file", path).Msg("failed to unmount")
		}
	}

	return os.RemoveAll(m.machineRoot(id))
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
			UID:            firecracker.Int(0),
			GID:            firecracker.Int(0),
			NumaNode:       firecracker.Int(0),
			ExecFile:       FCBin,
			JailerBinary:   JailerBin,
			ID:             vm.Name,
			ChrootBaseDir:  m.root,
			Daemonize:      true,
			ChrootStrategy: NewMountStrategy(m.machineRoot(vm.Name)),
			Stdout:         out,
			Stderr:         out,
		},
		Drives:         devices,
		ForwardSignals: []os.Signal{}, // it has to be an empty list to prevent using the default
	}

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
		m.Delete(vm.Name)
		return err
	}

	if err := machine.Start(ctx); err != nil {
		m.Delete(vm.Name)
		return err
	}

	return nil
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

func (m *vmModuleImpl) find(name string) (int, error) {
	const (
		proc   = "/proc"
		search = "/firecracker"
	)
	idArg := fmt.Sprintf("--id=%s", name)
	result := 0
	err := filepath.Walk(proc, func(path string, info os.FileInfo, err error) error {
		if path == proc {
			// assend into /proc
			return nil
		}

		dir, name := filepath.Split(path)

		if filepath.Clean(dir) != proc {
			// this to make sure we only scan first level
			return filepath.SkipDir
		}

		pid, err := strconv.Atoi(name)
		if err != nil {
			//not a number
			return nil //continue scan
		}
		cmd, err := ioutil.ReadFile(filepath.Join(path, "cmdline"))
		if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return err
		}

		parts := bytes.Split(cmd, []byte{0})
		if string(parts[0]) != search {
			return nil
		}

		// a firecracker instance, now find id
		for _, part := range parts {
			if string(part) == idArg {
				// a hit
				result = pid
				// this is to stop the scan.
				return errScanFound
			}
		}

		return nil
	})

	if err == errScanFound {
		return result, nil
	} else if err != nil {
		return 0, err
	}

	return 0, fmt.Errorf("vm '%s' not found", name)
}

func (m *vmModuleImpl) Delete(name string) error {
	defer m.cleanFs(name)

	if !m.exists(name) {
		return nil
	}

	pid, err := m.find(name)
	if err != nil {
		return err
	}

	client := firecracker.NewClient(m.socket(name), nil, false)
	// timeout is request timeout, not machine timeout to shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()
	action := models.InstanceActionInfoActionTypeSendCtrlAltDel
	info := models.InstanceActionInfo{
		ActionType: &action,
	}

	now := time.Now()

	const (
		termAfter = 5 * time.Second
		killAfter = 10 * time.Second
	)

	_, err = client.CreateSyncAction(ctx, &info)
	if err != nil {
		return err
	}

	for {
		if !m.exists(name) {
			return nil
		}

		if time.Since(now) > termAfter {
			syscall.Kill(pid, syscall.SIGTERM)
		}

		if time.Since(now) > killAfter {
			syscall.Kill(pid, syscall.SIGKILL)
			break
		}

		<-time.After(1 * time.Second)
	}

	return nil
}
