package vm

import (
	"context"
	"os"
	"path/filepath"
	"syscall"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/pkg/errors"
)

const (
	mountFilesToRootFSHandlerName = "fcinit.MountFilesToRootFS"
)

// MountStrategy is used by the jailer to link files from
// host to the vm chroot. This strategy uses a mount bind
type MountStrategy struct {
	chroot string
}

// NewMountStrategy creates a new mount strategy for machine root at root
func NewMountStrategy(root string) *MountStrategy {
	return &MountStrategy{chroot: filepath.Join(root, "root")}
}

// AdaptHandlers will inject the LinkFilesHandler into the handler list.
func (s *MountStrategy) AdaptHandlers(handlers *firecracker.Handlers) error {
	if !handlers.FcInit.Has(firecracker.SetupKernelArgsHandlerName) {
		return firecracker.ErrRequiredHandlerMissing
	}

	handlers.FcInit = handlers.FcInit.AppendAfter(
		firecracker.SetupKernelArgsHandlerName,
		s.handler(),
	)

	return nil
}

func (s *MountStrategy) handler() firecracker.Handler {
	return firecracker.Handler{
		Name: mountFilesToRootFSHandlerName,
		Fn: func(ctx context.Context, machine *firecracker.Machine) error {
			if err := os.MkdirAll(s.chroot, 0755); err != nil {
				return err
			}

			for _, cfg := range []*string{&machine.Cfg.KernelImagePath} {
				file := *cfg
				if len(file) == 0 {
					continue
				}

				// mount kernel
				if err := s.mount(file, s.rooted(file)); err != nil {
					return err
				}

				*cfg = filepath.Base(file)
			}

			// mount drives
			for i, drive := range machine.Cfg.Drives {
				name := filepath.Base(*drive.PathOnHost)
				if err := s.mount(*drive.PathOnHost, s.rooted(*drive.PathOnHost)); err != nil {
					return err
				}

				machine.Cfg.Drives[i].PathOnHost = firecracker.String(name)
			}

			return nil
		},
	}
}

func (s *MountStrategy) rooted(n string) string {
	return filepath.Join(s.chroot, filepath.Base(n))
}

func (s *MountStrategy) mount(src, dest string) error {
	if filepath.Clean(src) == filepath.Clean(dest) {
		// nothing to do here
		return nil
	}

	f, err := os.Create(dest)
	if err != nil {
		return errors.Wrapf(err, "failed to touch file: %s", dest)
	}

	f.Close()

	if err := syscall.Mount(src, dest, "", syscall.MS_BIND, ""); err != nil {
		return errors.Wrapf(err, "failed to mount '%s' > '%s'", src, dest)
	}

	return nil
}
