package zui

import (
	"context"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/threefoldtech/zbus"

	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	activeStatus     = "Active"
	InProgressStatus = "In progress"
	FailedStatus     = "Failed"
	InactiveStatus   = "Inactive"
)

func serviceRender(ctx context.Context, client zbus.Client, grid *ui.Grid, render *signalFlag) (chan bool, error) {
	services := widgets.NewTable()

	grid.Set(
		ui.NewRow(1,
			ui.NewCol(1, services),
		),
	)

	services.Title = "Services Status"
	services.RowSeparator = false

	services.Rows = [][]string{
		{"", "Status"},
		{"Networkd", InProgressStatus},
		{"Registerar", InactiveStatus},
		{"Statistics", InactiveStatus},
		{"Containerd", InactiveStatus},
		{"Storaged", InactiveStatus},
		{"Noded", InactiveStatus},
		{"Powerd", InactiveStatus},
	}

	render.Signal()

	done := make(chan bool)

	go func() {
		networkStatus := make(chan string)
		statisticsStatus := make(chan bool)
		containerdStatus := make(chan bool)
		storagedStatus := make(chan bool)
		nodedStatus := make(chan bool)
		powerdStatus := make(chan bool)

		go func() {
			networkStatus <- getNetworkStatus(ctx, client)
		}()

		go func() {
			getStatisticsStatus(ctx, client)
			statisticsStatus <- true
		}()

		go func() {
			getContainerdStatus(ctx, client)
			containerdStatus <- true
		}()
		go func() {
			getStoragedStatus(ctx, client)
			storagedStatus <- true
		}()
		go func() {
			getNodedStatus(ctx, client)
			nodedStatus <- true
		}()
		go func() {
			getPowerdStatus(ctx, client)
			powerdStatus <- true
		}()

		for i := 0; i < 6; i++ {
			services.Rows[2][1] = getRegistrarStatus(ctx, client)

			select {
			case status := <-networkStatus:
				services.Rows[1][1] = status
			case <-statisticsStatus:
				services.Rows[3][1] = green(activeStatus)
			case <-containerdStatus:
				services.Rows[4][1] = green(activeStatus)
			case <-storagedStatus:
				services.Rows[5][1] = green(activeStatus)
			case <-nodedStatus:
				services.Rows[6][1] = green(activeStatus)
			case <-powerdStatus:
				services.Rows[7][1] = green(activeStatus)
				render.Signal()
			}
		}

		done <- true
	}()
	return done, nil
}

func getRegistrarStatus(ctx context.Context, client zbus.Client) string {
	register := stubs.NewRegistrarStub(client)
	if _, err := register.NodeID(ctx); err != nil {
		if isInProgressError(err) {
			return InProgressStatus
		}
		return red(FailedStatus)
	}
	return green(activeStatus)
}

func getNetworkStatus(ctx context.Context, client zbus.Client) string {
	network := stubs.NewNetworkerStub(client)
	err := network.Ready(ctx)
	if err != nil {
		return red(FailedStatus)
	}
	return green(activeStatus)
}

func getStatisticsStatus(ctx context.Context, client zbus.Client) {
	statistics := stubs.NewStatisticsStub(client)
	statistics.Total(ctx)
}

func getContainerdStatus(ctx context.Context, client zbus.Client) {
	statistics := stubs.NewContainerModuleStub(client)
	statistics.ListNS(ctx)
}

func getStoragedStatus(ctx context.Context, client zbus.Client) {
	storaged := stubs.NewStorageModuleStub(client)
	storaged.Devices(ctx)
}

func getNodedStatus(ctx context.Context, client zbus.Client) {
	noded := stubs.NewSystemMonitorStub(client)
	noded.NodeID(ctx)
}

func getPowerdStatus(ctx context.Context, client zbus.Client) {
	powerd := stubs.NewIdentityManagerStub(client)
	powerd.NodeID(ctx)
}
