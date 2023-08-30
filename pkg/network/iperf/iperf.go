package iperf

import (
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	zinitService = "iperf"
)

// IPerfServer represent an iperf server
type IPerfServer struct {
}

// NewIPerfServer create a new iperf Server
func NewIPerfServer() *IPerfServer {
	return &IPerfServer{}
}

// Restart restarts iperf zinit service
func (s *IPerfServer) Restart(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGTERM)
}

// Reload reloads iperf zinit service
func (s *IPerfServer) Reload(z *zinit.Client) error {
	return z.Kill(zinitService, zinit.SIGHUP)
}

// Start creates an iperf zinit service and starts it
func (s *IPerfServer) Start(z *zinit.Client) error {
	// better if we just stop, forget and start over to make
	// sure we using the right exec params
	if _, err := z.Status(zinitService); err == nil {
		// not here we need to stop it
		if err := z.StopWait(5*time.Second, zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to stop iperf service")
		}
		if err := z.Forget(zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to forget iperf service")
		}
	}

	_, err := exec.LookPath("iperf")
	if err != nil {
		return err
	}

	cmd := `iperf -s`

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: cmd,
		After: []string{
			"node-ready",
		},
	})

	if err != nil {
		return errors.Wrap(err, "failed to add iperf service")
	}

	if err := z.Monitor(zinitService); err != nil && !errors.Is(err, zinit.ErrAlreadyMonitored) {
		return err
	}

	return z.StartWait(time.Second*20, zinitService)
}
