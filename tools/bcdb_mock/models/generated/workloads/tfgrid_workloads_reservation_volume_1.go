package workloads

import "encoding/json"

type TfgridWorkloadsReservationVolume1 struct {
	WorkloadId      int64                                        `bson:"workload_id" json:"workload_id"`
	NodeId          string                                       `bson:"node_id" json:"node_id"`
	Size            int64                                        `bson:"size" json:"size"`
	Type            TfgridWorkloadsReservationVolume1TypeEnum    `bson:"type" json:"type"`
	StatsAggregator []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationVolume1() (TfgridWorkloadsReservationVolume1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationVolume1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationVolume1TypeEnum uint8

const (
	TfgridWorkloadsReservationVolume1TypeHDD TfgridWorkloadsReservationVolume1TypeEnum = iota
	TfgridWorkloadsReservationVolume1TypeSSD
)

func (e TfgridWorkloadsReservationVolume1TypeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationVolume1TypeHDD:
		return "HDD"
	case TfgridWorkloadsReservationVolume1TypeSSD:
		return "SSD"
	}
	return "UNKNOWN"
}
