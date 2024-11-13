package zosapi

import (
	"context"

	"github.com/threefoldtech/zos/pkg/geoip"
)

func (g *ZosAPI) locationGet(ctx context.Context, payload []byte) (interface{}, error) {
	return geoip.Fetch()
}
