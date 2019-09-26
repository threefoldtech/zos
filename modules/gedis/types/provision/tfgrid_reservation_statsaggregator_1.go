package provision

import (
	"encoding/json"
	schema "github.com/threefoldtech/zosv2/modules/schema"
)

type TfgridReservationStatsaggregator1 struct {
	Addr   string `json:"addr"`
	Port   int64  `json:"port"`
	Secret string `json:"secret"`
}

func NewTfgridReservationStatsaggregator1() (TfgridReservationStatsaggregator1, error) {
	const value = "{}"
	var object TfgridReservationStatsaggregator1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridReservationResult1 struct {
	Category   TfgridReservationResult1CategoryEnum `json:"category"`
	WorkloadId int64                                `json:"workload_id"`
	DataJson   string                               `json:"data_json"`
	Signature  []byte                               `json:"signature"`
	State      TfgridReservationResult1StateEnum    `json:"state"`
	Message    string                               `json:"message"`
	Epoch      schema.Date                          `json:"epoch"`
}

func NewTfgridReservationResult1() (TfgridReservationResult1, error) {
	const value = "{\"message\": \"\"}"
	var object TfgridReservationResult1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridReservationResult1CategoryEnum uint8

const (
	TfgridReservationResult1CategoryZdb TfgridReservationResult1CategoryEnum = iota
	TfgridReservationResult1CategoryContainer
	TfgridReservationResult1CategoryNetwork
	TfgridReservationResult1CategoryVolume
)

func (e TfgridReservationResult1CategoryEnum) String() string {
	switch e {
	case TfgridReservationResult1CategoryZdb:
		return "zdb"
	case TfgridReservationResult1CategoryContainer:
		return "container"
	case TfgridReservationResult1CategoryNetwork:
		return "network"
	case TfgridReservationResult1CategoryVolume:
		return "volume"
	}
	return "UNKNOWN"
}

type TfgridReservationResult1StateEnum uint8

const (
	TfgridReservationResult1StateError TfgridReservationResult1StateEnum = iota
	TfgridReservationResult1StateOk
)

func (e TfgridReservationResult1StateEnum) String() string {
	switch e {
	case TfgridReservationResult1StateError:
		return "error"
	case TfgridReservationResult1StateOk:
		return "ok"
	}
	return "UNKNOWN"
}
