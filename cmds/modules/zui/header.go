package zui

import (
	"context"
	"fmt"

	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func green(s string) string {
	return fmt.Sprintf("[%s](fg:green)", s)
}

func red(s string) string {
	return fmt.Sprintf("[%s](fg:red)", s)
}

// func headerRenderer(c zbus.Client, h *widgets.Paragraph, r *Flag) error {
func headerRenderer(ctx context.Context, c zbus.Client, h *widgets.Paragraph, r *signalFlag) error {
	env, err := environment.Get()
	if err != nil {
		return err
	}

	identity := stubs.NewIdentityManagerStub(c)
	farmID, _ := identity.FarmID(ctx)

	h.Text = "\n    Fetching realtime node information... please wait."

	s := "          Welcome to [Zero-OS](fg:yellow), [ThreeFold](fg:blue) Autonomous Operating System\n" +
		"\n" +
		" This is node %s (farmer %s)\n" +
		" running Zero-OS version [%s](fg:blue) (mode [%s](fg:cyan))"

	host := stubs.NewVersionMonitorStub(c)
	ch, err := host.Version(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start update stream for version")
	}

	go func() {
		for version := range ch {
			var name string
			var nodeID string
			var farm string
			if name, err = identity.Farm(ctx); err != nil {
				farm = red(fmt.Sprintf("%d: %s", farmID, err.Error()))
			} else {
				farm = green(fmt.Sprintf("%d: %s", farmID, name))
			}

			if node, err := identity.NodeIDNumeric(ctx); err != nil {
				nodeID = red(err.Error())
			} else {
				nodeID = green(fmt.Sprint(node))
			}

			h.Text = fmt.Sprintf(s, nodeID, farm, version.String(), env.RunningMode.String())
			r.Signal()
		}
	}()

	return nil
}
