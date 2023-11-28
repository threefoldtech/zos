package zui

import (
	"context"
	"fmt"
	"sync"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/threefoldtech/zos/pkg"
)

type module struct {
	grid   *ui.Grid
	render Signaler
	labels []labelData
	table  *widgets.Table
	mu     *sync.Mutex
}

type Signaler interface {
	Signal()
}

type labelData struct {
	label  string
	errors []string
}

func New(ctx context.Context, grid *ui.Grid, render Signaler) pkg.ZUI {
	table := widgets.NewTable()
	grid.Set(
		ui.NewRow(1.0, table),
	)
	table.Title = "Errors"
	table.FillRow = true
	table.RowSeparator = false

	table.Rows = [][]string{
		{"[No Errors!](fg:green)"},
	}

	zuiModule := &module{
		grid:   grid,
		render: render,
		table:  table,
		labels: make([]labelData, 0),
		mu:     &sync.Mutex{},
	}
	go zuiModule.renderErrors(ctx)
	return zuiModule
}

var _ pkg.ZUI = (*module)(nil)

func (m *module) PushErrors(label string, errors []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, data := range m.labels {
		if data.label == label {
			m.labels[i].errors = errors
			return
		}
	}
	m.labels = append(m.labels, labelData{label, errors})
}

func (m *module) renderErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			m.mu.Lock()
			labels := make([]labelData, len(m.labels))
			copy(labels, m.labels)
			m.mu.Unlock()
			display(labels, m.table, m.render)
			// in case nothing got displayed
			<-time.After(2 * time.Second)
		}
	}
}

func display(labels []labelData, table *widgets.Table, render Signaler) {
	table.Rows = [][]string{
		{"[No Errors!](fg:green)"},
	}
	for _, label := range labels {
		for _, e := range label.errors {
			table.Rows = [][]string{
				{fmt.Sprintf("%s: [%s](fg:red)", label.label, e)},
			}
			render.Signal()
			<-time.After(2 * time.Second)
		}
	}

	render.Signal()
}
