package main

import (
	"context"
	"fmt"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func cpuRender(client zbus.Client, grid *ui.Grid, r func()) error {
	var (
		size = 100
	)

	cpus, err := cpu.Counts(true)
	if err != nil {
		return errors.Wrap(err, "failed to get CPU number")
	}

	plot := widgets.NewPlot()
	plot.Title = "Graph"
	plot.Data = make([][]float64, cpus)
	for i := range plot.Data {
		plot.Data[i] = make([]float64, size)
	}

	plot.MaxVal = 100
	plot.DataLabels = func() []string {
		var labels []string
		for i := 0; i < cpus; i++ {
			labels = append(labels, fmt.Sprintf("CPU %d", i))
		}
		return labels
	}()

	plot.Marker = widgets.MarkerDot
	plot.PlotType = widgets.ScatterPlot
	plot.LineColors = func() []ui.Color {
		var colors []ui.Color
		for i := 1; i <= cpus; i++ {
			colors = append(colors, ui.Color(i))
		}
		return colors
	}()

	table := widgets.NewTable()
	table.RowSeparator = false
	table.Title = "CPU"
	table.Rows = func() [][]string {
		names := make([][]string, cpus)
		for i := 0; i < cpus; i++ {
			table.RowStyles[i] = ui.Style{
				Fg: plot.LineColors[i],
			}

			names[i] = []string{
				fmt.Sprintf("CPU %d", i),
				"",
			}
		}

		return names
	}()

	grid.Set(
		ui.NewRow(1,
			ui.NewCol(8.0/10, plot),
			ui.NewCol(2.0/10, table),
		),
	)

	monitor := stubs.NewSystemMonitorStub(client)
	stream, err := monitor.CPU(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to start cpu monitor stream")
	}

	go func() {
		for point := range stream {
			size := plot.Size().X - 7 // the 7 is for frame and padding
			for i, cpu := range point {
				data := append(plot.Data[i], cpu.Percent)
				//size limit
				if len(data) > size {
					data = data[len(data)-size:]
				}

				plot.Data[i] = data
				table.Rows[i][1] = fmt.Sprintf("%0.00f%%", cpu.Percent)
			}
			r()
		}
	}()

	return nil
}
