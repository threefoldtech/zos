package directory

import (
	"encoding/json"
	schema "github.com/threefoldtech/zos/pkg/schema"
	"net"
)

type TfgridDirectoryNode2 struct {
	NodeId            string                             `bson:"node_id" json:"node_id"`
	NodeIdV1          string                             `bson:"node_id_v1" json:"node_id_v1"`
	FarmId            int64                              `bson:"farm_id" json:"farm_id"`
	OsVersion         string                             `bson:"os_version" json:"os_version"`
	Created           schema.Date                        `bson:"created" json:"created"`
	Updated           schema.Date                        `bson:"updated" json:"updated"`
	Uptime            int64                              `bson:"uptime" json:"uptime"`
	Address           string                             `bson:"address" json:"address"`
	Location          TfgridDirectoryLocation1           `bson:"location" json:"location"`
	TotalResources    TfgridDirectoryNodeResourceAmount1 `bson:"total_resources" json:"total_resources"`
	UsedResources     TfgridDirectoryNodeResourceAmount1 `bson:"used_resources" json:"used_resources"`
	ReservedResources TfgridDirectoryNodeResourceAmount1 `bson:"reserved_resources" json:"reserved_resources"`
	Proofs            []TfgridDirectoryNodeProof1        `bson:"proofs" json:"proofs"`
	Ifaces            []TfgridDirectoryNodeIface1        `bson:"ifaces" json:"ifaces"`
	PublicConfig      TfgridDirectoryNodePublicIface1    `bson:"public_config" json:"public_config"`
	ExitNode          bool                               `bson:"exit_node" json:"exit_node"`
	Approved          bool                               `bson:"approved" json:"approved"`
	PublicKeyHex      string                             `bson:"public_key_hex" json:"public_key_hex"`
	WgPorts           []int64                            `bson:"wg_ports" json:"wg_ports"`
}

func NewTfgridDirectoryNode2() (TfgridDirectoryNode2, error) {
	const value = "{\"approved\": false, \"public_key_hex\": \"\"}"
	var object TfgridDirectoryNode2
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeIface1 struct {
	Name    string           `bson:"name" json:"name"`
	Addrs   []schema.IPRange `bson:"addrs" json:"addrs"`
	Gateway []net.IP         `bson:"gateway" json:"gateway"`
}

func NewTfgridDirectoryNodeIface1() (TfgridDirectoryNodeIface1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeIface1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodePublicIface1 struct {
	Master  string                                  `bson:"master" json:"master"`
	Type    TfgridDirectoryNodePublicIface1TypeEnum `bson:"type" json:"type"`
	Ipv4    schema.IPRange                          `bson:"ipv4" json:"ipv4"`
	Ipv6    schema.IPRange                          `bson:"ipv6" json:"ipv6"`
	Gw4     net.IP                                  `bson:"gw4" json:"gw4"`
	Gw6     net.IP                                  `bson:"gw6" json:"gw6"`
	Version int64                                   `bson:"version" json:"version"`
}

func NewTfgridDirectoryNodePublicIface1() (TfgridDirectoryNodePublicIface1, error) {
	const value = "{}"
	var object TfgridDirectoryNodePublicIface1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeResourceAmount1 struct {
	Cru int64 `bson:"cru" json:"cru"`
	Mru int64 `bson:"mru" json:"mru"`
	Hru int64 `bson:"hru" json:"hru"`
	Sru int64 `bson:"sru" json:"sru"`
}

func NewTfgridDirectoryNodeResourceAmount1() (TfgridDirectoryNodeResourceAmount1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeResourceAmount1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeProof1 struct {
	Created      schema.Date            `bson:"created" json:"created"`
	HardwareHash string                 `bson:"hardware_hash" json:"hardware_hash"`
	DiskHash     string                 `bson:"disk_hash" json:"disk_hash"`
	Hardware     map[string]interface{} `bson:"hardware" json:"hardware"`
	Disks        map[string]interface{} `bson:"disks" json:"disks"`
}

func NewTfgridDirectoryNodeProof1() (TfgridDirectoryNodeProof1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeProof1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodePublicIface1TypeEnum uint8

const (
	TfgridDirectoryNodePublicIface1TypeMacvlan TfgridDirectoryNodePublicIface1TypeEnum = iota
	TfgridDirectoryNodePublicIface1TypeVlan
)

func (e TfgridDirectoryNodePublicIface1TypeEnum) String() string {
	switch e {
	case TfgridDirectoryNodePublicIface1TypeMacvlan:
		return "macvlan"
	case TfgridDirectoryNodePublicIface1TypeVlan:
		return "vlan"
	}
	return "UNKNOWN"
}
