package dhcp

import (
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/zinit"
)

// Probe is used to do some DHCP request on a interface
type Probe struct {
	cmd *exec.Cmd
}

// NewProbe returns a Probe
func NewProbe() *Probe {
	return &Probe{}
}

// Start starts the DHCP client process
func (d *Probe) Start(inf string) error {

	d.cmd = exec.Command("udhcpc",
		"-f", //foreground
		"-i", inf,
		"-t", "20", //send 20 dhcp queries
		"-T", "1", // every second
		"-s", "/usr/share/udhcp/simple.script",
		"-p", fmt.Sprintf("/run/udhcpc.%s.pid", inf),
		"--now", // exit if lease is not obtained
	)

	log.Debug().Msgf("start udhcp: %v", d.cmd.String())
	if err := d.cmd.Start(); err != nil {
		return err
	}

	return nil
}

// Run runs the DHCP client process and registers it to zinit
func (d *Probe) Run(inf string) error {
	z, err := zinit.New("")
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to zinit")
		return err
	}

	serviceName := fmt.Sprintf("dhcp-%s", inf)

	ns, err := exec.Command("ip", "netns", "identify").Output()
	if err != nil {
		return errors.Wrap(err, "failed to identify namespace")
	}

	err = zinit.AddService(serviceName, zinit.InitService{
		Exec:    fmt.Sprintf("ip netns exec %s /sbin/udhcpc -v -f -i %s -t 20 -T 1 -s /usr/share/udhcp/simple.script", ns, inf),
		Oneshot: false,
		After:   []string{},
	})

	if err != nil {
		log.Error().Err(err).Msg("fail to create dhcp-zos zinit service")
		return err
	}

	if err := z.Monitor(serviceName); err != nil {
		log.Error().Err(err).Msg("fail to start monitoring dhcp-zos zinit service")
		return err
	}

	if err := z.Start(serviceName); err != nil {
		log.Error().Err(err).Msg("fail to start dhcp-zos zinit service")
		return err
	}

	return nil
}

// Stop kills the DHCP client process
func (d *Probe) Stop() error {
	if d.cmd.ProcessState != nil && d.cmd.ProcessState.Exited() {
		return nil
	}

	if err := d.cmd.Process.Kill(); err != nil {
		log.Error().Err(err).Msg("could not kill proper")
		return err
	}

	_ = d.cmd.Wait()

	return nil
}
