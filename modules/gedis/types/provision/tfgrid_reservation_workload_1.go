package provision

import "encoding/json"

type TfgridReservationWorkload1 struct {
	WorkloadID string                             `json:"workload_id,omitempty"`
	Type       TfgridReservationWorkload1TypeEnum `json:"type,omitempty"`
	Workload   json.RawMessage                    `json:"content,omitempty"`
	User       string                             `json:"user,omitempty"`
	Created    int64                              `json:"created,omitempty"`
	Duration   int64                              `json:"duration,omitempty"`
	Signature  string                             `json:"signature,omitempty"`
	ToDelete   bool                               `json:"to_delete,omitempty"`
}

func NewTfgridReservationWorkload1() (TfgridReservationWorkload1, error) {
	const value = "{}"
	var object TfgridReservationWorkload1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridReservationWorkload1TypeEnum uint8

const (
	TfgridReservationWorkload1TypeNetwork TfgridReservationWorkload1TypeEnum = iota
	TfgridReservationWorkload1TypeVolume
	TfgridReservationWorkload1TypeZdb
	TfgridReservationWorkload1TypeContainer
)

func (e TfgridReservationWorkload1TypeEnum) String() string {
	switch e {
	case TfgridReservationWorkload1TypeNetwork:
		return "network"
	case TfgridReservationWorkload1TypeVolume:
		return "volume"
	case TfgridReservationWorkload1TypeZdb:
		return "zdb"
	case TfgridReservationWorkload1TypeContainer:
		return "container"
	}
	return "UNKNOWN"
}
