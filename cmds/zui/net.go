package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/network"
	"github.com/threefoldtech/zos/pkg/stubs"
)

func netRender(client zbus.Client, grid *ui.Grid, r func()) error {
	rx := widgets.NewSparkline()
	rx.LineColor = ui.ColorBlue

	tx := widgets.NewSparkline()
	tx.LineColor = ui.ColorMagenta

	net := widgets.NewSparklineGroup(rx, tx)
	net.Title = "Network"

	info := widgets.NewParagraph()
	info.Title = "Addresses"

	grid.Set(
		ui.NewRow(1.0/3,
			ui.NewCol(1, info),
		),
		ui.NewRow(2.0/3,
			ui.NewCol(1, net),
		),
	)

	monitor := stubs.NewSystemMonitorStub(client)
	host := stubs.NewHostMonitorStub(client)
	ctx := context.Background()
	stats, err := monitor.Nics(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start net monitor stream")
	}

	address, err := host.IPs(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to start ip monitor stream")
	}

	var buffer strings.Builder

	var sent uint64
	var recv uint64
	t := time.Now()

	go func() {
		for {
			select {
			case a := <-address:
				buffer.Reset()
				for _, ip := range a {
					if buffer.Len() > 0 {
						buffer.WriteByte('\n')
					}
					buffer.WriteString(ip.String())
				}
				info.Text = buffer.String()
			case s := <-stats:
				for _, nic := range s {
					if nic.Name != network.DefaultBridge {
						continue
					}
					size := net.Size().X - 2
					if sent == 0 || recv == 0 {
						sent = nic.BytesSent
						recv = nic.BytesRecv
						continue
					}

					//nic.
					now := time.Now()
					delta := float64(now.Sub(t)) / float64(time.Second)
					rrate := float64(nic.BytesRecv-recv) / delta
					trate := float64(nic.BytesSent-sent) / delta
					sent = nic.BytesSent
					recv = nic.BytesRecv
					t = now

					rrate = rrate / 1024 // KB
					trate = trate / 1024 // KB

					rx.Title = fmt.Sprintf("%0.00f KB", rrate)
					tx.Title = fmt.Sprintf("%0.00f KB", trate)

					rx.Data = append(rx.Data, rrate)
					tx.Data = append(tx.Data, trate)

					rx.Data = trimFloat64(rx.Data, size)
					tx.Data = trimFloat64(tx.Data, size)
				}
			}

			r()
		}
	}()

	return nil
}
