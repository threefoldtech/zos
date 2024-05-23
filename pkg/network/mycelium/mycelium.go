package mycelium

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/zinit"
)

const (
	zinitService = "mycelium"
	tunName      = "utun9"
	confPath     = "/var/cache/modules/networkd/mycelium.conf"
	keyFile      = "var/cache/modules/networkd/mycelium_priv_key.bin"
)

func EnsureMyceluim(ns string) error {
	cfg, err := environment.GetConfig()
	if err != nil {
		return err
	}

	cli := zinit.Default()
	peers := cfg.Mycelium.Peers

	// better if we just stop, forget and start over to make
	// sure we using the right exec params
	if _, err := cli.Status(zinitService); err == nil {
		if err := cli.StopWait(5*time.Second, zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to stop mycelium service")
		}
		if err := cli.Forget(zinitService); err != nil && !errors.Is(err, zinit.ErrUnknownService) {
			return errors.Wrap(err, "failed to forget mycelium service")
		}
	}

	bin, err := exec.LookPath("mycelium")
	if err != nil {
		return err
	}

	cmd := `sh -c '
		exec ip netns exec %s %s --key-file %s --tun-name %s --peers %s
	'`

	err = zinit.AddService(zinitService, zinit.InitService{
		Exec: fmt.Sprintf(cmd, ns, bin, confPath, tunName, strings.Join(peers, " ")),
	})
	if err != nil {
		return err
	}

	return cli.Monitor(zinitService)
}
