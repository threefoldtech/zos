package migrate

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
)

const (
	imageURL = "https://bootstrap.grid.tf/uefimg/%s/%d"
)

var (
	modeName = map[environment.RunningMode]string{
		environment.RunningDev:  "dev",
		environment.RunningTest: "test",
		environment.RunningMain: "prod",
	}
)

func burn(ctx context.Context, v3 uint64, device *Device) error {
	env, err := environment.Get()
	if err != nil {
		return errors.Wrap(err, "failed to get environment")
	}

	dev, err := os.OpenFile(device.Path, os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return errors.Wrap(err, "failed to open device for writing")
	}

	defer dev.Close()

	url := fmt.Sprintf(imageURL, modeName[env.RunningMode], v3)
	response, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to get zos v3 image")
	}

	defer func() {
		ioutil.ReadAll(response.Body)
		response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return fmt.Errorf("invalid response '%s': %s", response.Status, string(body))
	}

	// this is un recoverable, if the node reboots now while the image is being written
	// it can corrupt the usb stick.
	_, err = io.Copy(dev, response.Body)
	if err != nil {
		return errors.Wrap(err, "failed to write image to device")
	}
	return nil
}

func wipe(ctx context.Context) error {
	devices, err := devices(ctx, Not(IsUsb))
	if err != nil {
		return err
	}

	for _, device := range devices {
		if len(device.Mountpoint) != 0 {
			// yes we don't care about the unmounting, if it fails np.
			// we continue with the wiping anyway
			if err := syscall.Unmount(device.Mountpoint, syscall.MNT_FORCE|syscall.MNT_DETACH); err != nil {
				log.Error().Err(err).Msg("unmounting device '%s'")
			}
		}

		log.Info().Str("device", device.Path).Msg("wiping device")
		if err := exec.CommandContext(ctx, "wipefs", "-a", "-f", device.Path).Run(); err != nil {
			log.Error().Err(err).Msg("failed to run wipefs on device")
		}
	}

	return nil
}
