package workloads

import "encoding/json"

type TfgridWorkloadsReservationZdb1 struct {
	WorkloadId      int64                                        `bson:"workload_id" json:"workload_id"`
	NodeId          string                                       `bson:"node_id" json:"node_id"`
	Size            int64                                        `bson:"size" json:"size"`
	Mode            TfgridWorkloadsReservationZdb1ModeEnum       `bson:"mode" json:"mode"`
	Password        string                                       `bson:"password" json:"password"`
	DiskType        TfgridWorkloadsReservationZdb1DiskTypeEnum   `bson:"disk_type" json:"disk_type"`
	Public          bool                                         `bson:"public" json:"public"`
	StatsAggregator []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationZdb1() (TfgridWorkloadsReservationZdb1, error) {
	const value = "{\"public\": false}"
	var object TfgridWorkloadsReservationZdb1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationZdb1ModeEnum uint8

const (
	TfgridWorkloadsReservationZdb1ModeSeq TfgridWorkloadsReservationZdb1ModeEnum = iota
	TfgridWorkloadsReservationZdb1ModeUser
)

func (e TfgridWorkloadsReservationZdb1ModeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationZdb1ModeSeq:
		return "seq"
	case TfgridWorkloadsReservationZdb1ModeUser:
		return "user"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservationZdb1DiskTypeEnum uint8

const (
	TfgridWorkloadsReservationZdb1DiskTypeHdd TfgridWorkloadsReservationZdb1DiskTypeEnum = iota
	TfgridWorkloadsReservationZdb1DiskTypeSsd
)

func (e TfgridWorkloadsReservationZdb1DiskTypeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationZdb1DiskTypeHdd:
		return "hdd"
	case TfgridWorkloadsReservationZdb1DiskTypeSsd:
		return "ssd"
	}
	return "UNKNOWN"
}
