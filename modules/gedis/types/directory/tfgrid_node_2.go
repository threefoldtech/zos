package directory

import (
	"encoding/json"
	schema "github.com/threefoldtech/zosv2/modules/schema"
	"net"
)

type TfgridNode2 struct {
	NodeId            string                    `json:"node_id"`
	FarmId            string                    `json:"farm_id"`
	OsVersion         string                    `json:"os_version"`
	Created           schema.Date               `json:"created"`
	Updated           schema.Date               `json:"updated"`
	Uptime            int64                     `json:"uptime"`
	Address           string                    `json:"address"`
	Location          TfgridLocation1           `json:"location"`
	TotalResources    TfgridNodeResourceAmount1 `json:"total_resources"`
	UsedResources     TfgridNodeResourceAmount1 `json:"used_resources"`
	ReservedResources TfgridNodeResourceAmount1 `json:"reserved_resources"`
	Proofs            []TfgridNodeProof1        `json:"proofs"`
	Ifaces            []TfgridNodeIface1        `json:"ifaces"`
	PublicConfig      TfgridNodePublicIface1    `json:"public_config"`
	ExitNode          bool                      `json:"exit_node"`
	Approved          bool                      `json:"approved"`
	PublicKeyHex      string                    `json:"public_key_hex"`
}

func NewTfgridNode2() (TfgridNode2, error) {
	const value = "{\"approved\": false, \"public_key_hex\": \"\"}"
	var object TfgridNode2
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNodeIface1 struct {
	Name    string           `json:"name"`
	Addrs   []schema.IPRange `json:"addrs"`
	Gateway []net.IP         `json:"gateway"`
}

func NewTfgridNodeIface1() (TfgridNodeIface1, error) {
	const value = "{}"
	var object TfgridNodeIface1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNodePublicIface1 struct {
	Master  string                         `json:"master"`
	Type    TfgridNodePublicIface1TypeEnum `json:"type"`
	Ipv4    net.IP                         `json:"ipv4"`
	Ipv6    net.IP                         `json:"ipv6"`
	Gw4     net.IP                         `json:"gw4"`
	Gw6     net.IP                         `json:"gw6"`
	Version int64                          `json:"version"`
}

func NewTfgridNodePublicIface1() (TfgridNodePublicIface1, error) {
	const value = "{}"
	var object TfgridNodePublicIface1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNodeResourceAmount1 struct {
	Cru int64 `json:"cru"`
	Mru int64 `json:"mru"`
	Hru int64 `json:"hru"`
	Sru int64 `json:"sru"`
}

func NewTfgridNodeResourceAmount1() (TfgridNodeResourceAmount1, error) {
	const value = "{}"
	var object TfgridNodeResourceAmount1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNodeProof1 struct {
	Created      schema.Date            `json:"created"`
	HardwareHash string                 `json:"hardware_hash"`
	DiskHash     string                 `json:"disk_hash"`
	Hardware     map[string]interface{} `json:"hardware"`
	Disks        map[string]interface{} `json:"disks"`
}

func NewTfgridNodeProof1() (TfgridNodeProof1, error) {
	const value = "{}"
	var object TfgridNodeProof1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNodePublicIface1TypeEnum uint8

const (
	TfgridNodePublicIface1TypeMacvlan TfgridNodePublicIface1TypeEnum = iota
)

func (e TfgridNodePublicIface1TypeEnum) String() string {
	switch e {
	case TfgridNodePublicIface1TypeMacvlan:
		return "macvlan"
	}
	return "UNKNOWN"
}
