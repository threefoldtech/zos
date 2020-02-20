package workloads

import (
	"encoding/json"
	schema "github.com/threefoldtech/zos/pkg/schema"
)

type TfgridWorkloadsReservation1 struct {
	Json                string                                        `bson:"json" json:"json"`
	DataReservation     TfgridWorkloadsReservationData1               `bson:"data_reservation" json:"data_reservation"`
	CustomerTid         int64                                         `bson:"customer_tid" json:"customer_tid"`
	CustomerSignature   string                                        `bson:"customer_signature" json:"customer_signature"`
	NextAction          TfgridWorkloadsReservation1NextActionEnum     `bson:"next_action" json:"next_action"`
	SignaturesProvision []TfgridWorkloadsReservationSigningSignature1 `bson:"signatures_provision" json:"signatures_provision"`
	SignaturesFarmer    []TfgridWorkloadsReservationSigningSignature1 `bson:"signatures_farmer" json:"signatures_farmer"`
	SignaturesDelete    []TfgridWorkloadsReservationSigningSignature1 `bson:"signatures_delete" json:"signatures_delete"`
	Epoch               schema.Date                                   `bson:"epoch" json:"epoch"`
	Results             []TfgridWorkloadsReservationResult1           `bson:"results" json:"results"`
}

func NewTfgridWorkloadsReservation1() (TfgridWorkloadsReservation1, error) {
	const value = "{\"json\": \"\"}"
	var object TfgridWorkloadsReservation1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationData1 struct {
	Description             string                                    `bson:"description" json:"description"`
	SigningRequestProvision TfgridWorkloadsReservationSigningRequest1 `bson:"signing_request_provision" json:"signing_request_provision"`
	SigningRequestDelete    TfgridWorkloadsReservationSigningRequest1 `bson:"signing_request_delete" json:"signing_request_delete"`
	Containers              []TfgridWorkloadsReservationContainer1    `bson:"containers" json:"containers"`
	Volumes                 []TfgridWorkloadsReservationVolume1       `bson:"volumes" json:"volumes"`
	Zdbs                    []TfgridWorkloadsReservationZdb1          `bson:"zdbs" json:"zdbs"`
	Networks                []TfgridWorkloadsReservationNetwork1      `bson:"networks" json:"networks"`
	Kubernetes              []TfgridWorkloadsReservationK8S1          `bson:"kubernetes" json:"kubernetes"`
	ExpirationProvisioning  schema.Date                               `bson:"expiration_provisioning" json:"expiration_provisioning"`
	ExpirationReservation   schema.Date                               `bson:"expiration_reservation" json:"expiration_reservation"`
}

func NewTfgridWorkloadsReservationData1() (TfgridWorkloadsReservationData1, error) {
	const value = "{\"description\": \"\"}"
	var object TfgridWorkloadsReservationData1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationSigningRequest1 struct {
	Signers   []int64 `bson:"signers" json:"signers"`
	QuorumMin int64   `bson:"quorum_min" json:"quorum_min"`
}

func NewTfgridWorkloadsReservationSigningRequest1() (TfgridWorkloadsReservationSigningRequest1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationSigningRequest1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationSigningSignature1 struct {
	Tid       int64       `bson:"tid" json:"tid"`
	Signature string      `bson:"signature" json:"signature"`
	Epoch     schema.Date `bson:"epoch" json:"epoch"`
}

func NewTfgridWorkloadsReservationSigningSignature1() (TfgridWorkloadsReservationSigningSignature1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationSigningSignature1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservation1NextActionEnum uint8

const (
	TfgridWorkloadsReservation1NextActionCreate TfgridWorkloadsReservation1NextActionEnum = iota
	TfgridWorkloadsReservation1NextActionSign
	TfgridWorkloadsReservation1NextActionPay
	TfgridWorkloadsReservation1NextActionDeploy
	TfgridWorkloadsReservation1NextActionDelete
	TfgridWorkloadsReservation1NextActionInvalid
	TfgridWorkloadsReservation1NextActionDeleted
)

func (e TfgridWorkloadsReservation1NextActionEnum) String() string {
	switch e {
	case TfgridWorkloadsReservation1NextActionCreate:
		return "create"
	case TfgridWorkloadsReservation1NextActionSign:
		return "sign"
	case TfgridWorkloadsReservation1NextActionPay:
		return "pay"
	case TfgridWorkloadsReservation1NextActionDeploy:
		return "deploy"
	case TfgridWorkloadsReservation1NextActionDelete:
		return "delete"
	case TfgridWorkloadsReservation1NextActionInvalid:
		return "invalid"
	case TfgridWorkloadsReservation1NextActionDeleted:
		return "deleted"
	}
	return "UNKNOWN"
}
