package workloads

import (
	"encoding/json"
	schema "github.com/threefoldtech/zos/pkg/schema"
)

type TfgridWorkloadsReservationResult1 struct {
	Category   TfgridWorkloadsReservationResult1CategoryEnum `bson:"category" json:"category"`
	WorkloadId string                                        `bson:"workload_id" json:"workload_id"`
	DataJson   string                                        `bson:"data_json" json:"data_json"`
	Signature  []byte                                        `bson:"signature" json:"signature"`
	State      TfgridWorkloadsReservationResult1StateEnum    `bson:"state" json:"state"`
	Message    string                                        `bson:"message" json:"message"`
	Epoch      schema.Date                                   `bson:"epoch" json:"epoch"`
}

func NewTfgridWorkloadsReservationResult1() (TfgridWorkloadsReservationResult1, error) {
	const value = "{\"message\": \"\"}"
	var object TfgridWorkloadsReservationResult1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationResult1CategoryEnum uint8

const (
	TfgridWorkloadsReservationResult1CategoryZdb TfgridWorkloadsReservationResult1CategoryEnum = iota
	TfgridWorkloadsReservationResult1CategoryContainer
	TfgridWorkloadsReservationResult1CategoryNetwork
	TfgridWorkloadsReservationResult1CategoryVolume
)

func (e TfgridWorkloadsReservationResult1CategoryEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationResult1CategoryZdb:
		return "zdb"
	case TfgridWorkloadsReservationResult1CategoryContainer:
		return "container"
	case TfgridWorkloadsReservationResult1CategoryNetwork:
		return "network"
	case TfgridWorkloadsReservationResult1CategoryVolume:
		return "volume"
	}
	return "UNKNOWN"
}

type TfgridWorkloadsReservationResult1StateEnum uint8

const (
	TfgridWorkloadsReservationResult1StateError TfgridWorkloadsReservationResult1StateEnum = iota
	TfgridWorkloadsReservationResult1StateOk
	TfgridWorkloadsReservationResult1StateDeleted
)

func (e TfgridWorkloadsReservationResult1StateEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationResult1StateError:
		return "error"
	case TfgridWorkloadsReservationResult1StateOk:
		return "ok"
	case TfgridWorkloadsReservationResult1StateDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}
