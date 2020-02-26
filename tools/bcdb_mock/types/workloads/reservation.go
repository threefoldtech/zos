package workloads

import (
	generated "github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
)

type Reservation generated.TfgridWorkloadsReservation1

// Pipeline changes Reservation R as defined by the reservation pipeline
// returns new reservation object, and true if the reservation has changed
type Pipeline struct {
	R Reservation
}

// Next gets new modified reservation, and true if the reservation has changed from the input
func (p *Pipeline) Next() (Reservation, bool) {

	return p.R, false
}
