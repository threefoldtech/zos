package iperf

import (
	"os/exec"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	zinitService = "iperf"
)

// Ensure creates an iperf zinit service and monitors it
func Ensure(z *zinit.Client) error {
	if _, err := z.Status(zinitService); err == nil {
		return nil
	}

	_, err := exec.LookPath("iperf")
	if err != nil {
		return err
	}

	cmd := `ip netns exec public iperf -s`

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: cmd,
		After: []string{
			"networkd",
		},
	})

	if err != nil {
		return errors.Wrap(err, "failed to add iperf service")
	}

	return z.Monitor(zinitService)
}
