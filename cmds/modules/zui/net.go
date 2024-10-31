package zui

import (
	"context"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	_ "github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos4/pkg"
	"github.com/threefoldtech/zos4/pkg/netlight/types"
	"github.com/threefoldtech/zos4/pkg/stubs"
)

func addressRender(ctx context.Context, table *widgets.Table, client zbus.Client, render *signalFlag) error {
	table.Title = "Network"
	table.FillRow = true
	table.RowSeparator = false

	table.Rows = [][]string{
		{"ZOS", loading},
		{"DUL", loading},
	}
	table.Rows[1][1] = "single" // light has only single stack setup
	stub := stubs.NewNetworkerLightStub(client)
	zos, err := stub.ZOSAddresses(ctx)
	if err != nil {
		return err
	}

	toString := func(al pkg.NetlinkAddresses) string {
		var buf strings.Builder
		for _, a := range al {
			if a.IP == nil || len(a.IP) == 0 {
				continue
			}

			if buf.Len() > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(a.String())
		}

		return buf.String()
	}

	a, err := stub.Interfaces(ctx, types.DefaultBridge, "")
	if err != nil {
		return err
	}

	table.Rows[0][1] = toString(a.Interfaces[types.DefaultBridge].IPs)

	go func() {
		for {
			render.Signal()
			table.ColumnWidths = []int{6, table.Size().X - 9}
			table.Rows[0][1] = toString(<-zos)

		}
	}()

	return nil
}

func netRender(client zbus.Client, grid *ui.Grid, render *signalFlag) error {
	addresses := widgets.NewTable()

	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1, addresses),
		),
	)
	ctx := context.Background()

	if err := addressRender(ctx, addresses, client, render); err != nil {
		return err
	}

	return nil
}
