package zosapi

import (
	"context"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/geoip"
)

func (g *ZosAPI) locationGet(ctx context.Context, payload []byte) (interface{}, error) {
	if _, err := os.Stat(geoip.LocationFile); err != nil {
		return nil, errors.Wrap(err, "couldn't found a location info")
	}

	f, err := os.Open(geoip.LocationFile)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get the location info")
	}
	defer f.Close()

	var loc geoip.Location
	err = json.NewDecoder(f).Decode(&loc)
	return loc, err
}
