package provision

import "encoding/json"

type TfgridReservationVolume1 struct {
	WorkloadId      int64                               `json:"workload_id"`
	NodeId          int64                               `json:"node_id"`
	ReservationId   int64                               `json:"reservation_id"`
	Size            int64                               `json:"size"`
	Type            TfgridReservationVolume1TypeEnum    `json:"type"`
	StatsAggregator []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	FarmerTid       int64                               `json:"farmer_tid"`
}

func NewTfgridReservationVolume1() (TfgridReservationVolume1, error) {
	const value = "{}"
	var object TfgridReservationVolume1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridReservationVolume1TypeEnum uint8

const (
	TfgridReservationVolume1TypeHDD TfgridReservationVolume1TypeEnum = iota
	TfgridReservationVolume1TypeSSD
)

func (e TfgridReservationVolume1TypeEnum) String() string {
	switch e {
	case TfgridReservationVolume1TypeHDD:
		return "HDD"
	case TfgridReservationVolume1TypeSSD:
		return "SSD"
	}
	return "UNKNOWN"
}
