package dhcp

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/zinit"
)

// Probe is used to do some DHCP request on a interface
type Probe struct {
	cmd *exec.Cmd
}

// BackgroundProbe is used to do some DHCP request on a interface controlled by zinit
type BackgroundProbe struct {
	cmd *exec.Cmd
	z   *zinit.Client
}

// NewProbe returns a Probe
func NewProbe() *Probe {
	return &Probe{}
}

// NewBackgroundProbe return a new background Probe that can be controlled with zinit
func NewBackgroundProbe() (*BackgroundProbe, error) {
	z, err := zinit.New("")
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to zinit")
		return nil, err
	}
	return &BackgroundProbe{
		z: z,
	}, nil
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

// Start runs the DHCP client process and registers it to zinit
func (d *BackgroundProbe) Start(inf string) error {
	serviceName := fmt.Sprintf("dhcp-%s", inf)

	ns, err := exec.Command("ip", "netns", "identify").Output()
	if err != nil {
		return errors.Wrap(err, "failed to identify namespace")
	}

	exec := fmt.Sprintf("/sbin/udhcpc -v -f -i %s -t 20 -T 1 -s /usr/share/udhcp/simple.script", inf)

	if strings.Trim(string(ns), " ") != "" {
		exec = fmt.Sprintf("ip netns exec %s %s", ns, exec)
	}

	err = zinit.AddService(serviceName, zinit.InitService{
		Exec:    exec,
		Oneshot: false,
		After:   []string{},
	})

	if err != nil {
		log.Error().Err(err).Msg("fail to create dhcp-zos zinit service")
		return err
	}

	if err := d.z.Monitor(serviceName); err != nil {
		log.Error().Err(err).Msg("fail to start monitoring dhcp-zos zinit service")
		return err
	}

	return nil
}

// IsRunning checks if a background process is running in zinit
func (d *BackgroundProbe) IsRunning(inf string) (bool, error) {
	serviceName := fmt.Sprintf("dhcp-%s", inf)

	services, err := d.z.List()
	if err != nil {
		log.Error().Err(err).Msg("fail to create dhcp-zos zinit service")
		return false, err
	}

	if _, ok := services[serviceName]; ok {
		return true, nil
	}
	return false, nil
}

// Stop stops a zinit background process
func (d *BackgroundProbe) Stop(inf string) error {
	serviceName := fmt.Sprintf("dhcp-%s", inf)
	return d.z.Stop(serviceName)
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
