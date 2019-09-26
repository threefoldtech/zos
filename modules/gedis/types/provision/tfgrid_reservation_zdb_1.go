package provision

import "encoding/json"

type TfgridReservationZdb1 struct {
	WorkloadId      int64                               `json:"workload_id"`
	NodeId          int64                               `json:"node_id"`
	ReservationId   int64                               `json:"reservation_id"`
	Size            int64                               `json:"size"`
	Mode            TfgridReservationZdb1ModeEnum       `json:"mode"`
	Password        string                              `json:"password"`
	DiskType        TfgridReservationZdb1DiskTypeEnum   `json:"disk_type"`
	Public          bool                                `json:"public"`
	StatsAggregator []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	FarmerTid       int64                               `json:"farmer_tid"`
}

func NewTfgridReservationZdb1() (TfgridReservationZdb1, error) {
	const value = "{\"public\": false}"
	var object TfgridReservationZdb1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridReservationZdb1ModeEnum uint8

const (
	TfgridReservationZdb1ModeSeq TfgridReservationZdb1ModeEnum = iota
	TfgridReservationZdb1ModeUser
)

func (e TfgridReservationZdb1ModeEnum) String() string {
	switch e {
	case TfgridReservationZdb1ModeSeq:
		return "seq"
	case TfgridReservationZdb1ModeUser:
		return "user"
	}
	return "UNKNOWN"
}

type TfgridReservationZdb1DiskTypeEnum uint8

const (
	TfgridReservationZdb1DiskTypeHdd TfgridReservationZdb1DiskTypeEnum = iota
	TfgridReservationZdb1DiskTypeSsd
)

func (e TfgridReservationZdb1DiskTypeEnum) String() string {
	switch e {
	case TfgridReservationZdb1DiskTypeHdd:
		return "hdd"
	case TfgridReservationZdb1DiskTypeSsd:
		return "ssd"
	}
	return "UNKNOWN"
}
