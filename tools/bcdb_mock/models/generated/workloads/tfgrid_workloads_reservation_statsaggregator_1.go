package workloads

import "encoding/json"

type TfgridWorkloadsReservationStatsaggregator1 struct {
	Addr   string `bson:"addr" json:"addr"`
	Port   int64  `bson:"port" json:"port"`
	Secret string `bson:"secret" json:"secret"`
}

func NewTfgridWorkloadsReservationStatsaggregator1() (TfgridWorkloadsReservationStatsaggregator1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationStatsaggregator1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
