package explorer

import (
	"github.com/threefoldtech/tfexplorer/models/generated/workloads"
	"github.com/threefoldtech/zos/pkg/provision"
)

// ReservationConverterFunc is used to convert from the explorer workloads type into the
// internal Reservation type
type ReservationConverterFunc func(w workloads.Workloader) (*provision.Reservation, error)

//ResultConverterFunc is used to convert internal Result type to the explorer workload result
type ResultConverterFunc func(result provision.Result) (*workloads.Result, error)
