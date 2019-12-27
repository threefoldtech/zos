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
	usedPlot := widgets.NewSparkline()

	usedPlot.MaxVal = 100
	usedPlot.LineColor = ui.ColorRed

	used := widgets.NewSparklineGroup(usedPlot)
	used.Title = "Used Memory"
	freePlot := widgets.NewSparkline()

	freePlot.MaxVal = 100
	freePlot.LineColor = ui.ColorGreen

	free := widgets.NewSparklineGroup(freePlot)
	free.Title = "Free Memory"
	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1./2, used),
			ui.NewCol(1./2, free),
		),
	)

	monitor := stubs.NewSystemMonitorStub(client)
	stream, err := monitor.Memory(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start mem monitor stream")
	}

	go func() {
		for point := range stream {
			size := used.Size().X - 2
			data := append(usedPlot.Data, point.UsedPercent)
			if len(data) > size {
				data = data[len(data)-size:]
			}
			usedPlot.Data = data
			size = free.Size().X - 2
			data = append(freePlot.Data, 100-point.UsedPercent)
			if len(data) > size {
				data = data[len(data)-size:]
			}
			freePlot.Data = data
			r()
		}
	}()

	return nil
}
