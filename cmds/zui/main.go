package main

import (
	"flag"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/version"
)

func trimFloat64(a []float64, size int) []float64 {
	if len(a) > size {
		return a[len(a)-size:]
	}
	return a
}

func main() {
	app.Initialize()

	var (
		msgBrokerCon string
		ver          bool
	)

	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	client, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to zbus")
	}

	if err := ui.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize term ui")
	}

	defer ui.Close()

	width, height := ui.TerminalDimensions()
	header := widgets.NewParagraph()
	header.Border = false
	grid := ui.NewGrid()

	headerHeight := 3
	header.SetRect(0, -1, width, headerHeight)
	//header.Text = "ZeroOS"
	header.TextStyle = ui.Style{
		Fg:       ui.ColorBlue,
		Bg:       ui.ColorClear,
		Modifier: ui.ModifierBold,
	}
	grid.Title = "System"

	grid.SetRect(0, headerHeight-2, width, height)

	cpu := ui.NewGrid()
	cpu.Title = "CPU"
	cpu.Border = true

	mem := ui.NewGrid()
	mem.Title = "Memory"
	mem.Border = true

	net := ui.NewGrid()
	net.Title = "Networ"
	net.Border = true

	grid.Set(
		ui.NewRow(1.0/4,
			ui.NewCol(1, cpu),
		),
		ui.NewRow(1.0/8,
			ui.NewCol(1, mem),
		),
		ui.NewRow(1.0/2,
			ui.NewCol(1.0/2, net),
			// add provision info here.
		),
	)

	render := func() {
		ui.Render(header, grid)
	}

	if err := headerRenderer(client, header, render); err != nil {
		log.Error().Err(err).Msg("failed to start header renderer")
	}
	if err := cpuRender(client, cpu, render); err != nil {
		log.Error().Err(err).Msg("failed to start cpu renderer")
	}
	if err := memRender(client, mem, render); err != nil {
		log.Error().Err(err).Msg("failed to start mem renderer")
	}

	if err := netRender(client, net, render); err != nil {
		log.Error().Err(err).Msg("failed to start net renderer")
	}

	ui.Clear()
	render()

	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				header.SetRect(0, 0, payload.Width, headerHeight)
				grid.SetRect(0, headerHeight, payload.Width, payload.Height-headerHeight)
				ui.Clear()
				render()
			}
		}
	}
}
