package vm

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// Run run the machine with cloud-hypervisor
func (m *Machine) Run(ctx context.Context, socket, logs string) error {

	// build command line
	args := []string{
		"cloud-hypervisor",
		"--kernel", m.Boot.Kernel,
		"--initramfs", m.Boot.Initrd,
		"--cmdline", m.Boot.Args,

		"--cpus", m.Config.CPU.String(),
		"--memory", m.Config.Mem.String(),

		"--log-file", logs,
		"--api-socket", socket,
	}

	// disks
	if len(m.Disks) > 0 {
		args = append(args, "--disk")
		for _, disk := range m.Disks {
			args = append(args, disk.String())
		}
	}

	if len(m.Interfaces) > 0 {
		args = append(args, "--net")
		for _, nic := range m.Interfaces {
			args = append(args, nic.String())
		}
	}

	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, arg))
	}

	cmd := exec.CommandContext(ctx,
		"ash", "-c",
		fmt.Sprintf("%s &", strings.Join(quoted, " ")),
	)

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to start cloud-hypervisor")
	}

	return nil
}
