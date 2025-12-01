package vmtestd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosbase/pkg"
	"github.com/threefoldtech/zosbase/pkg/gridtypes"
	"github.com/threefoldtech/zosbase/pkg/stubs"
	"github.com/threefoldtech/zosbase/pkg/utils"
)

const module = "vmtest"

// Module is vmtestd entry point
var Module cli.Command = cli.Command{
	Name:  "vmtestd",
	Usage: "periodically deploys and decommissions test VMs",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "root",
			Usage: "`ROOT` working directory of the module",
			Value: "/var/cache/modules/vmtestd",
		},
		&cli.StringFlag{
			Name:  "broker",
			Usage: "connection string to the message `BROKER`",
			Value: "unix:///var/run/redis.sock",
		},
		&cli.DurationFlag{
			Name:  "interval",
			Usage: "interval between VM test deployments",
			Value: 2 * time.Hour,
		},
	},
	Action: action,
}

// VMTestService manages VM test deployments
type VMTestService struct {
	cl       zbus.Client
	root     string
	interval time.Duration
}

// NewVMTestService creates a new VM test service
func NewVMTestService(cl zbus.Client, root string, interval time.Duration) *VMTestService {
	return &VMTestService{
		cl:       cl,
		root:     root,
		interval: interval,
	}
}

// Run starts the VM test service loop
func (s *VMTestService) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Info().Msgf("VM test service started with interval: %s", s.interval)

	// Run immediately on start
	if err := s.runTest(ctx); err != nil {
		log.Error().Err(err).Msg("initial VM test failed")
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("VM test service stopping")
			return ctx.Err()
		case <-ticker.C:
			if err := s.runTest(ctx); err != nil {
				log.Error().Err(err).Msg("VM test failed")
			}
		}
	}
}

// runTest deploys a test VM, waits, then decommissions it
func (s *VMTestService) runTest(ctx context.Context) error {
	log.Info().Msg("starting VM test deployment")

	vmd := stubs.NewVMModuleStub(s.cl)
	flist := stubs.NewFlisterStub(s.cl)

	// Create a test VM ID
	vmID := fmt.Sprintf("test-vm-%d", time.Now().Unix())
	flistURL := "https://hub.threefold.me/tf-official-apps/redis_zinit.flist"

	log.Info().Str("vm_id", vmID).Str("flist", flistURL).Msg("deploying test VM")

	// Deploy the VM
	deployCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// List VMs to verify connection
	vms, err := vmd.List(deployCtx)
	if err != nil {
		return errors.Wrap(err, "failed to connect to VM module")
	}

	log.Info().Int("existing_vms", len(vms)).Msg("VM module connection successful")

	// Mount the flist to inspect its contents
	log.Info().Str("flist", flistURL).Msg("mounting flist")
	mountPath, err := flist.Mount(deployCtx, vmID, flistURL, pkg.MountOptions{
		ReadOnly: true,
	})
	if err != nil {
		return errors.Wrap(err, "failed to mount flist")
	}

	// Ensure we unmount the flist when done
	defer func() {
		if unmountErr := flist.Unmount(context.Background(), vmID); unmountErr != nil {
			log.Error().Err(unmountErr).Str("vm_id", vmID).Msg("failed to unmount flist")
		}
	}()

	log.Info().Str("mount_path", mountPath).Msg("flist mounted successfully")

	// Get flist info (kernel, initrd, image paths)
	flistInfo, err := s.getFlistInfo(mountPath)
	if err != nil {
		return errors.Wrap(err, "failed to get flist info")
	}

	log.Info().
		Bool("is_container", flistInfo.IsContainer()).
		Str("kernel", flistInfo.KernelPath).
		Str("initrd", flistInfo.InitrdPath).
		Str("image", flistInfo.ImagePath).
		Msg("flist info retrieved")

	// Create VM configuration
	vmConfig := pkg.VM{
		Name:        vmID,
		CPU:         1,
		Memory:      gridtypes.Unit(512 * gridtypes.Megabyte),
		Network:     pkg.VMNetworkInfo{},
		NoKeepAlive: false,
	}

	// Configure boot based on flist type
	if flistInfo.IsContainer() {
		// Container mode - boot from virtio-fs
		log.Info().Msg("configuring as container VM")
		if len(flistInfo.KernelPath) != 0 {
			vmConfig.KernelImage = flistInfo.KernelPath
			vmConfig.InitrdImage = flistInfo.InitrdPath
		} else {
			// Use default cloud-hypervisor kernel
			vmConfig.KernelImage = "/usr/lib/kernel/vmlinuz"
			vmConfig.InitrdImage = "/usr/lib/kernel/initrd.img"
		}
		vmConfig.Boot = pkg.Boot{
			Type: pkg.BootVirtioFS,
			Path: mountPath,
		}
	} else {
		// VM mode - boot from disk image
		log.Info().Msg("configuring as full VM with disk image")
		vmConfig.KernelImage = flistInfo.KernelPath
		if len(flistInfo.InitrdPath) != 0 {
			vmConfig.InitrdImage = flistInfo.InitrdPath
		}
		vmConfig.Boot = pkg.Boot{
			Type: pkg.BootDisk,
			Path: flistInfo.ImagePath,
		}
	}

	log.Info().Str("vm_id", vmID).Str("flist", vmConfig.Boot.Path).Msg("deploying VM with flist")

	// Deploy the VM
	machineInfo, err := vmd.Run(deployCtx, vmConfig)
	if err != nil {
		return errors.Wrap(err, "failed to deploy VM")
	}

	log.Info().
		Str("vm_id", vmID).
		Str("console_url", machineInfo.ConsoleURL).
		Msg("test VM deployed successfully")

	// Wait a bit to let the VM run
	time.Sleep(10 * time.Second)

	// Decommission the VM
	log.Info().Str("vm_id", vmID).Msg("decommissioning test VM")

	decommissionCtx, cancelDecommission := context.WithTimeout(ctx, 1*time.Minute)
	defer cancelDecommission()

	if err := vmd.Delete(decommissionCtx, vmID); err != nil {
		return errors.Wrap(err, "failed to decommission VM")
	}

	log.Info().Str("vm_id", vmID).Msg("test VM decommissioned successfully")

	return nil
}

func action(cli *cli.Context) error {
	var (
		moduleRoot   string        = cli.String("root")
		msgBrokerCon string        = cli.String("broker")
		interval     time.Duration = cli.Duration("interval")
	)

	if err := os.MkdirAll(moduleRoot, 0750); err != nil {
		return errors.Wrap(err, "fail to create module root")
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		return errors.Wrap(err, "fail to connect to message broker")
	}

	service := NewVMTestService(client, moduleRoot, interval)

	log.Info().
		Str("broker", msgBrokerCon).
		Str("root", moduleRoot).
		Dur("interval", interval).
		Msg("starting vmtestd module")

	ctx, _ := utils.WithSignal(context.Background())
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	if err := service.Run(ctx); err != nil && err != context.Canceled {
		return errors.Wrap(err, "unexpected error")
	}

	return nil
}

// FListInfo contains virtual machine flist details
type FListInfo struct {
	ImagePath  string
	KernelPath string
	InitrdPath string
}

// IsContainer returns true if this is a container (no disk image)
func (f *FListInfo) IsContainer() bool {
	return len(f.ImagePath) == 0
}

// getFlistInfo inspects a mounted flist and extracts kernel, initrd, and image paths
func (s *VMTestService) getFlistInfo(flistPath string) (flist FListInfo, err error) {
	files := map[string]*string{
		"/image.raw":       &flist.ImagePath,
		"/boot/vmlinuz":    &flist.KernelPath,
		"/boot/initrd.img": &flist.InitrdPath,
	}

	for rel, ptr := range files {
		path := filepath.Join(flistPath, rel)

		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return flist, errors.Wrapf(err, "couldn't stat %s", rel)
		}

		if stat.IsDir() {
			return flist, fmt.Errorf("path '%s' cannot be a directory", rel)
		}

		mod := stat.Mode()
		switch mod.Type() {
		case 0:
			// regular file, do nothing
		case os.ModeSymlink:
			// this is a symlink, validate it points inside the flist
			link, err := os.Readlink(path)
			if err != nil {
				return flist, errors.Wrapf(err, "failed to read link at '%s", rel)
			}
			// the link if joined with path (and cleaned) must point to somewhere under flistPath
			abs := filepath.Clean(filepath.Join(flistPath, link))
			if !strings.HasPrefix(abs, flistPath) {
				return flist, fmt.Errorf("path '%s' points to invalid location", rel)
			}
		default:
			return flist, fmt.Errorf("path '%s' is of invalid type: %s", rel, mod.Type().String())
		}

		// set the value
		*ptr = path
	}

	return flist, nil
}
