package vm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Run run the machine with cloud-hypervisor
func (m *Machine) Run(ctx context.Context, socket, logs string) error {

	// build command line
	args := map[string][]string{
		"--kernel":    {m.Boot.Kernel},
		"--initramfs": {m.Boot.Initrd},
		"--cmdline":   {m.Boot.Args},

		"--cpus":   {m.Config.CPU.String()},
		"--memory": {m.Config.Mem.String()},

		"--log-file":   {logs},
		"--api-socket": {socket},
	}

	// disks
	if len(m.Disks) > 0 {
		var disks []string
		for _, disk := range m.Disks {
			disks = append(disks, disk.String())
		}
		args["--disk"] = disks
	}

	if len(m.Interfaces) > 0 {
		var interfaces []string
		for _, nic := range m.Interfaces {
			interfaces = append(interfaces, nic.String())
		}
		args["--net"] = interfaces
	}

	tmp, err := ioutil.TempFile("", "vm-exec-*.sh")
	if err != nil {
		return errors.Wrap(err, "failed to create vm exec script")
	}

	log.Debug().Str("name", m.ID).Str("exec", tmp.Name()).Msg("using exec file")
	defer func() {
		tmp.Close()
		//os.Remove(tmp.Name())
	}()

	const debug = false
	if debug {
		args["--console"] = []string{"off"}
		args["--serial"] = []string{"tty"}
	}

	// todo: check write error
	tmp.WriteString("exec cloud-hypervisor")

	for k, vs := range args {
		tmp.WriteString(" \\\n\t")
		tmp.WriteString(k)
		for _, v := range vs {
			tmp.WriteString(" ")
			tmp.WriteString("'")
			tmp.WriteString(v)
			tmp.WriteString("'")
		}
	}
	tmp.WriteString("\n")
	tmp.Close()

	log.Debug().Str("name", m.ID).Msg("starting machine")
	//the reason we do this shit is that we want the process to daemoinize the process in the back ground
	var cmd *exec.Cmd
	if debug {
		cmd = exec.CommandContext(ctx, "ash", tmp.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
	} else {
		cmd = exec.CommandContext(ctx, "ash", "-c", fmt.Sprintf("ash %s > %s 2>&1 &", tmp.Name(), logs+".out"))
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to start cloud-hypervisor")
	}

	return nil
}
