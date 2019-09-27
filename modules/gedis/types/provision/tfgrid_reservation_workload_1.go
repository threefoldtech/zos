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

type TfgridReservationWorkload1TypeEnum uint8

func (e *TfgridReservationWorkload1TypeEnum) UnmarshalJSON(b []byte) error {
	var i uint8
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	*e = TfgridReservationWorkload1TypeEnum(i)
	return nil
}

const (
	TfgridReservationWorkload1TypeZdb TfgridReservationWorkload1TypeEnum = iota
	TfgridReservationWorkload1TypeContainer
	TfgridReservationWorkload1TypeVolume
	TfgridReservationWorkload1TypeNetwork
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
