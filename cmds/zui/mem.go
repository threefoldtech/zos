package main

import (
	"context"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func memRender(client zbus.Client, grid *ui.Grid, r func()) error {

	plot := widgets.NewSparkline()
	plot.Title = "Memory"

	plot.MaxVal = 100
	plot.LineColor = ui.ColorGreen

	w := widgets.NewSparklineGroup(plot)
	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1, w),
		),
	)

	monitor := stubs.NewSystemMonitorStub(client)
	stream, err := monitor.Memory(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start mem monitor stream")
	}

	go func() {
		for point := range stream {
			size := w.Size().X - 2
			data := append(plot.Data, point.UsedPercent)
			if len(data) > size {
				data = data[len(data)-size:]
			}
			plot.Data = data
			r()
		}
	}()

	return nil
}
