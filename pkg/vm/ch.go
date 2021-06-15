package vm

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	chBin = "cloud-hypervisor"
)

// Run run the machine with cloud-hypervisor
func (m *Machine) Run(ctx context.Context, socket, logs string) error {

	// build command line
	args := map[string][]string{
		"--kernel":  {m.Boot.Kernel},
		"--cmdline": {m.Boot.Args},

		"--cpus":   {m.Config.CPU.String()},
		"--memory": {fmt.Sprintf("%s,shared=on", m.Config.Mem.String())},

		"--log-file":   {logs},
		"--console":    {"off"},
		"--serial":     {fmt.Sprintf("file=%s.console", logs)},
		"--api-socket": {socket},
	}
	var err error
	var pids []int
	defer func() {
		if err != nil {
			for _, pid := range pids {
				syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	}()

	var filesystems []string
	for i, fs := range m.FS {
		socket := filepath.Join("/var", "run", fmt.Sprintf("virtio-%s-%d.socket", m.ID, i))
		var pid int
		pid, err = m.startFs(socket, fs.Path)
		if err != nil {
			return err
		}
		pids = append(pids, pid)
		filesystems = append(filesystems, fmt.Sprintf("tag=%s,socket=%s", fs.Tag, socket))
	}

	if len(filesystems) > 0 {
		args["--fs"] = filesystems
	}

	if m.Boot.Initrd != "" {
		args["--initramfs"] = []string{m.Boot.Initrd}
	}
	// disks
	if len(m.Disks) > 0 {
		var disks []string
		for _, disk := range m.Disks {
			disks = append(disks, disk.String())
		}
		args["--disk"] = disks
	}

	var fds []int
	if len(m.Interfaces) > 0 {
		var interfaces []string

		for _, nic := range m.Interfaces {
			var typ InterfaceType
			var idx int
			typ, idx, err = nic.getType()
			if err != nil {
				return errors.Wrapf(err, "failed to detect interface type '%s'", nic.Tap)
			}
			if typ == InterfaceTAP {
				interfaces = append(interfaces, nic.asTap())
			} else if typ == InterfaceMACvTAP {
				// macvtap
				fd := len(fds) + 3
				fds = append(fds, idx)
				// fds[fd] = idx
				interfaces = append(interfaces, nic.asMACvTap(fd))
			} else {
				err = fmt.Errorf("unsupported tap device type '%s'", nic.Tap)
				return err
			}
		}
		args["--net"] = interfaces
	}

	const debug = false
	if debug {
		args["--serial"] = []string{"tty"}
	}

	var argsList []string
	for k, vl := range args {
		argsList = append(argsList, k)
		argsList = append(argsList, vl...)
	}

	var fullArgs []string
	// setting setsid
	// without this the CH process will exit if vmd is stopped.
	// optimally, this should be done by the SysProcAttr
	// but we always get permission denied error and it's not
	// clear why. so for now we use busybox setsid command to do
	// this.
	fullArgs = append(fullArgs, "setsid", chBin)
	fullArgs = append(fullArgs, argsList...)
	log.Debug().Msgf("ch: %+v", fullArgs)
	cmd := exec.CommandContext(ctx, "busybox", fullArgs...)
	// TODO: always get permission denied when setting
	// sid with sys proc attr
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Setsid:     true,
	// 	Setpgid:    true,
	// 	Foreground: false,
	// 	Noctty:     true,
	// 	Setctty:    true,
	// }

	var toClose []io.Closer

	for _, tapindex := range fds {
		var tap *os.File
		tap, err = os.OpenFile(filepath.Join("/dev", fmt.Sprintf("tap%d", tapindex)), os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		toClose = append(toClose, tap)
		cmd.ExtraFiles = append(cmd.ExtraFiles, tap)
	}

	defer func() {
		for _, c := range toClose {
			c.Close()
		}
	}()

	if err = cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start cloud-hypervisor")
	}

	if err = m.release(cmd.Process); err != nil {
		return err
	}

	return nil
}

func (m *Machine) startFs(socket, path string) (int, error) {
	cmd := exec.Command("busybox", "setsid",
		"virtiofsd-rs",
		"--socket", socket,
		"--shared-dir", path,
	)

	if err := cmd.Start(); err != nil {
		return 0, errors.Wrap(err, "failed to start virtiofsd-")
	}

	return cmd.Process.Pid, m.release(cmd.Process)
}

func (m *Machine) release(ps *os.Process) error {
	pid := ps.Pid
	go func() {
		ps, err := os.FindProcess(pid)
		if err != nil {
			log.Error().Err(err).Msgf("failed to find process with id: %d", pid)
			return
		}

		ps.Wait()
	}()

	if err := ps.Release(); err != nil {
		return errors.Wrap(err, "failed to release cloud-hypervisor process")
	}

	return nil
}
