package workloads

import (
	"encoding/json"
	schema "github.com/threefoldtech/zos/pkg/schema"
)

type TfgridWorkloadsReservationWorkload1 struct {
	WorkloadId string                                      `bson:"workload_id" json:"workload_id"`
	User       string                                      `bson:"user" json:"user"`
	Type       TfgridWorkloadsReservationWorkload1TypeEnum `bson:"type" json:"type"`
	Content    map[string]interface{}                      `bson:"content" json:"content"`
	Created    schema.Date                                 `bson:"created" json:"created"`
	Duration   int64                                       `bson:"duration" json:"duration"`
	Signature  string                                      `bson:"signature" json:"signature"`
	ToDelete   bool                                        `bson:"to_delete" json:"to_delete"`
}

func NewTfgridWorkloadsReservationWorkload1() (TfgridWorkloadsReservationWorkload1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationWorkload1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationWorkload1TypeEnum uint8

const (
	TfgridWorkloadsReservationWorkload1TypeZdb TfgridWorkloadsReservationWorkload1TypeEnum = iota
	TfgridWorkloadsReservationWorkload1TypeContainer
	TfgridWorkloadsReservationWorkload1TypeVolume
	TfgridWorkloadsReservationWorkload1TypeNetwork
	TfgridWorkloadsReservationWorkload1TypeKubernetes
)

func (e TfgridWorkloadsReservationWorkload1TypeEnum) String() string {
	switch e {
	case TfgridWorkloadsReservationWorkload1TypeZdb:
		return "zdb"
	case TfgridWorkloadsReservationWorkload1TypeContainer:
		return "container"
	case TfgridWorkloadsReservationWorkload1TypeVolume:
		return "volume"
	case TfgridWorkloadsReservationWorkload1TypeNetwork:
		return "network"
	case TfgridWorkloadsReservationWorkload1TypeKubernetes:
		return "kubernetes"
	}
	return "UNKNOWN"
}
