package provision

import (
	"encoding/json"

	schema "github.com/threefoldtech/zosv2/modules/schema"
)

type TfgridReservationNetwork1 struct {
	Name             string                              `json:"name"`
	WorkloadId       int64                               `json:"workload_id"`
	Iprange          schema.IPRange                      `json:"iprange"`
	StatsAggregator  []TfgridReservationStatsaggregator1 `json:"stats_aggregator"`
	NetworkResources []TfgridNetworkNetResource1         `json:"network_resources"`
}

func NewTfgridReservationNetwork1() (TfgridReservationNetwork1, error) {
	const value = "{\"name\": \"\", \"iprange\": \"10.10.0.0/16\"}"
	var object TfgridReservationNetwork1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNetworkNetResource1 struct {
	NodeId                       string           `json:"node_id"`
	Prefix                       schema.IPRange   `json:"prefix"`
	WireguardPrivateKeyEncrypted string           `json:"wireguard_private_key_encrypted"`
	WireguardPublicKey           string           `json:"wireguard_public_key"`
	Peers                        []WireguardPeer1 `json:"peers"`
}

func NewTfgridNetworkNetResource1() (TfgridNetworkNetResource1, error) {
	const value = "{\"wireguard_private_key_encrypted\": \"\", \"wireguard_public_key\": \"\"}"
	var object TfgridNetworkNetResource1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type WireguardPeer1 struct {
	PublicKey string `json:"public_key"`
	Endpoint  int64  `json:"endpoint"`
}

func NewWireguardPeer1() (WireguardPeer1, error) {
	const value = "{\"public_key\": \"\", \"endpoint\": \"\"}"
	var object WireguardPeer1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
