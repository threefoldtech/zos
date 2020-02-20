package workloads

import (
	"encoding/json"
	schema "github.com/threefoldtech/zos/pkg/schema"
)

type TfgridWorkloadsReservationNetwork1 struct {
	Name             string                                       `bson:"name" json:"name"`
	WorkloadId       int64                                        `bson:"workload_id" json:"workload_id"`
	Iprange          schema.IPRange                               `bson:"iprange" json:"iprange"`
	StatsAggregator  []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	NetworkResources []TfgridWorkloadsNetworkNetResource1         `bson:"network_resources" json:"network_resources"`
	FarmerTid        int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationNetwork1() (TfgridWorkloadsReservationNetwork1, error) {
	const value = "{\"name\": \"\", \"iprange\": \"10.10.0.0/16\"}"
	var object TfgridWorkloadsReservationNetwork1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsNetworkNetResource1 struct {
	NodeId                       string                          `bson:"node_id" json:"node_id"`
	WireguardPrivateKeyEncrypted string                          `bson:"wireguard_private_key_encrypted" json:"wireguard_private_key_encrypted"`
	WireguardPublicKey           string                          `bson:"wireguard_public_key" json:"wireguard_public_key"`
	WireguardListenPort          int64                           `bson:"wireguard_listen_port" json:"wireguard_listen_port"`
	Iprange                      schema.IPRange                  `bson:"iprange" json:"iprange"`
	Peers                        []TfgridWorkloadsWireguardPeer1 `bson:"peers" json:"peers"`
}

func NewTfgridWorkloadsNetworkNetResource1() (TfgridWorkloadsNetworkNetResource1, error) {
	const value = "{\"wireguard_private_key_encrypted\": \"\", \"wireguard_public_key\": \"\", \"iprange\": \"10.10.10.0/24\"}"
	var object TfgridWorkloadsNetworkNetResource1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsWireguardPeer1 struct {
	PublicKey      string           `bson:"public_key" json:"public_key"`
	AllowedIprange []schema.IPRange `bson:"allowed_iprange" json:"allowed_iprange"`
	Endpoint       string           `bson:"endpoint" json:"endpoint"`
	Iprange        schema.IPRange   `bson:"iprange" json:"iprange"`
}

func NewTfgridWorkloadsWireguardPeer1() (TfgridWorkloadsWireguardPeer1, error) {
	const value = "{\"public_key\": \"\", \"allowed_iprange\": [], \"endpoint\": \"\", \"iprange\": \"10.10.11.0/24\"}"
	var object TfgridWorkloadsWireguardPeer1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
