package network

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules"
)

type watcher struct {
	netID modules.NetID
	db    TNoDB
}

func NewWatcher(netID modules.NetID, db TNoDB) *watcher {
	return &watcher{
		netID: netID,
		db:    db,
	}
}

func (w *watcher) Watch(ctx context.Context) (<-chan *modules.Network, error) {
	nw, err := w.db.GetNetwork(w.netID)
	if err != nil {
		return nil, errors.Wrapf(err, "network watcher fail to get network %s", w.netID)
	}
	version := nw.Version

	ch := make(chan *modules.Network)
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Minute * 5):
			}

			nw, err := w.db.GetNetwork(w.netID)
			if err != nil {
				log.Error().
					Str("network", string(w.netID)).
					Err(err).Msg("fail to watch network")
				continue
			}

			if nw.Version > version {
				log.Info().
					Str("network", string(w.netID)).
					Uint32("current", version).
					Uint32("new", nw.Version).
					Msg("new network version found")
				version = nw.Version

				select {
				case <-ctx.Done():
					break
				case ch <- nw:
				}
			}
		}
	}()

	return ch, nil
}
