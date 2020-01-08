package main

import (
	"context"
	"fmt"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func addressRender(ctx context.Context, table *widgets.Table, client zbus.Client, render func()) error {
	table.Title = "Addresses"
	table.FillRow = true
	table.RowSeparator = false

	table.Rows = [][]string{
		{"ZOS", "Not Configured"},
		{"DMZ", "Not Configured"},
		{"Public", "Not Configured"},
	}

	stub := stubs.NewNetworkerStub(client)
	zos, err := stub.ZOSAddresses(ctx)
	if err != nil {
		return err
	}
	dmz, err := stub.DMZAddresses(ctx)
	if err != nil {
		return err
	}

	pub, err := stub.PublicAddresses(ctx)
	if err != nil {
		return err
	}

	toString := func(al pkg.NetlinkAddresses) string {
		var buf strings.Builder
		for _, a := range al {
			if buf.Len() > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(a.String())
		}

		return buf.String()
	}

	go func() {
		for {
			table.ColumnWidths = []int{6, table.Size().X - 9}
			select {
			case a := <-zos:
				table.Rows[0][1] = toString(a)
			case a := <-dmz:
				table.Rows[1][1] = toString(a)
			case a := <-pub:
				table.Rows[2][1] = toString(a)
			}

			render()
		}
	}()

	return nil
}

func netRender(client zbus.Client, grid *ui.Grid, render func()) error {
	addresses := widgets.NewTable()
	statistics := widgets.NewTable()

	statistics.Rows = [][]string{
		{"NIC", "SENT", "RECV"},
	}

	grid.Set(
		ui.NewRow(1.0/2,
			ui.NewCol(1, addresses),
		),
		ui.NewRow(1.0/2,
			ui.NewCol(1, statistics),
		),
	)

	ctx := context.Background()

	if err := addressRender(ctx, addresses, client, render); err != nil {
		return err
	}

	monitor := stubs.NewSystemMonitorStub(client)
	stats, err := monitor.Nics(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	go func() {
		for s := range stats {
			if len(statistics.Rows) != len(s)+1 {
				rows := [][]string{
					[]string{"NIC", "SENT", "RECV"},
				}
				for i := 0; i < len(s); i++ {
					rows = append(rows, make([]string, 3))
				}

				statistics.Rows = rows
			}

			rows := statistics.Rows
			for i, nic := range s {
				rows[i+1][0] = nic.Name
				rows[i+1][1] = fmt.Sprintf("%d KB", nic.RateOut/1024)
				rows[i+1][2] = fmt.Sprintf("%d KB", nic.RateIn/1024)
			}

			render()
		}
	}()

	return nil
}
