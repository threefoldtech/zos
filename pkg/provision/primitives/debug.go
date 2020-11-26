package primitives

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/zinit"

	"github.com/pkg/errors"
)

// Debug provision schema
type Debug struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Channel string `json:"channel"`
}

func (p *Primitives) debugProvision(ctx context.Context, reservation *provision.Reservation) (interface{}, error) {
	var cfg Debug
	if err := json.Unmarshal(reservation.Data, &cfg); err != nil {
		return nil, err
	}

	_, err := p.startZLF(ctx, reservation.ID, cfg)
	// nothing to return to BCDB
	return nil, err
}

func (p *Primitives) debugDecommission(ctx context.Context, reservation *provision.Reservation) error {
	return p.stopZLF(ctx, reservation.ID)
}

func (p *Primitives) startZLF(ctx context.Context, ID string, cfg Debug) (string, error) {
	identity := stubs.NewIdentityManagerStub(p.zbus)

	path, err := exec.LookPath("zlf")
	if err != nil {
		return "", errors.Wrap(err, "failed to start zlf")
	}

	z, err := zinit.New("")
	if err != nil {
		return "", errors.Wrap(err, "fail to connect to zinit")
	}
	defer z.Close()

	channel := fmt.Sprintf("%s-logs", identity.NodeID().Identity())
	if cfg.Channel != "" {
		channel = cfg.Channel
	}

	s := zinit.InitService{
		Exec:    fmt.Sprintf("%s --host %s --port %d --channel %s", path, cfg.Host, cfg.Port, channel),
		Oneshot: false,
		After:   []string{"networkd"},
		Log:     zinit.StdoutLogType,
	}

	name := fmt.Sprintf("zlf-debug-%s", ID)
	if err := zinit.AddService(name, s); err != nil {
		return "", errors.Wrap(err, "fail to add init service to zinit")
	}

	if err := z.Monitor(name); err != nil {
		return "", errors.Wrap(err, "failed to start monitoring zlf service")
	}

	return name, nil
}

func (p *Primitives) stopZLF(ctx context.Context, ID string) error {
	z, err := zinit.New("")
	if err != nil {
		return errors.Wrap(err, "fail to connect to zinit")
	}
	defer z.Close()

	name := fmt.Sprintf("zlf-debug-%s", ID)
	services, err := z.List()
	if err != nil {
		return errors.Wrap(err, "failed to list zinit services")
	}
	found := false
	for s := range services {
		if strings.Contains(s, name) {
			found = true
			break
		}
	}
	if !found {
		log.Info().Str("service", name).Msg("zinit service not found, nothing else to do")
		return nil
	}

	if err := z.Stop(name); err != nil {
		return errors.Wrapf(err, "failed to stop %s zlf service", name)
	}

	if err := z.Forget(name); err != nil {
		return errors.Wrapf(err, "failed to forget %s zlf service", name)
	}

	if err := zinit.RemoveService(name); err != nil {
		return errors.Wrapf(err, "failed to delete %s zlf service", name)
	}

	return nil
}
