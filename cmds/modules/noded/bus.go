package noded

import (
	"context"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/capacity"
	"github.com/threefoldtech/zos/pkg/rmb"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func rmbApi(ctx context.Context, cl zbus.Client, broker string) error {

	oracle := capacity.NewResourceOracle(stubs.NewStorageModuleStub(cl))

	dmi, err := oracle.DMI()
	if err != nil {
		return errors.Wrap(err, "failed to get dmi information")
	}

	hypervisor, err := oracle.GetHypervisor()
	if err != nil {
		return errors.Wrap(err, "failed to get hypervisors")
	}

	bus, err := rmb.New(broker)
	if err != nil {
		return errors.Wrap(err, "failed to initialize message bus server")
	}

	bus.WithHandler("zos.system.version", func(ctx context.Context, payload []byte) (interface{}, error) {
		ver := stubs.NewVersionMonitorStub(cl)
		output, err := exec.CommandContext(ctx, "zinit", "-V").CombinedOutput()
		var zInitVer string
		if err != nil {
			zInitVer = err.Error()
		} else {
			zInitVer = strings.TrimSpace(strings.TrimPrefix(string(output), "zinit"))
		}

		version := struct {
			ZOS   string `json:"zos"`
			ZInit string `json:"zinit"`
		}{
			ZOS:   ver.GetVersion(ctx).String(),
			ZInit: zInitVer,
		}

		return version, nil
	})

	bus.WithHandler("zos.system.dmi", func(ctx context.Context, payload []byte) (interface{}, error) {
		return dmi, nil
	})

	bus.WithHandler("zos.system.hypervisor", func(ctx context.Context, payload []byte) (interface{}, error) {
		return hypervisor, nil
	})

	// answer calls for dmi
	go func() {
		if err := bus.Run(ctx); err != nil {
			log.Fatal().Err(err).Msg("message bus handler failure")
		}
	}()

	return nil
}
