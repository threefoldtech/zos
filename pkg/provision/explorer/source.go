package explorer

import (
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfexplorer/client"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/provision"
)

type ReservationPoller struct {
	wl             client.Workloads
	inputConv      provision.ReservationConverterFunc
	provisionOrder map[provision.ReservationType]int
}

func (r *ReservationPoller) Poll(nodeID pkg.Identifier, from uint64) ([]*provision.Reservation, error) {

	list, err := r.wl.Workloads(nodeID.Identity(), from)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving workloads from explorer: %w", err)
	}

	result := make([]*provision.Reservation, 0, len(list))
	for _, wl := range list {
		log.Info().Msgf("convert %+v", wl)
		r, err := r.inputConv(wl)
		if err != nil {
			return nil, err
		}

		result = append(result, r)
	}

	if r.provisionOrder != nil {
		// sorts the workloads in the oder they need to be processed by provisiond
		sort.Slice(result, func(i int, j int) bool {
			return r.provisionOrder[result[i].Type] < r.provisionOrder[result[j].Type]
		})
	}

	return result, nil
}

// ReservationPollerFromWorkloads returns a reservation poller from client.Workloads
func ReservationPollerFromWorkloads(wl client.Workloads, inputConv provision.ReservationConverterFunc, provisionOrder map[provision.ReservationType]int) *ReservationPoller {
	return &ReservationPoller{
		wl:             wl,
		inputConv:      inputConv,
		provisionOrder: provisionOrder,
	}
}
