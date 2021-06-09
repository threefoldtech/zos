package mbus

import (
	"context"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/provision/mw"
)

func (a *WorkloadsMessagebus) getStatistics(ctx context.Context) (interface{}, mw.Response) {
	return struct {
		Total gridtypes.Capacity `json:"total"`
		Used  gridtypes.Capacity `json:"used"`
	}{
		Total: a.stats.Total(),
		Used:  a.stats.Current(),
	}, nil
}
