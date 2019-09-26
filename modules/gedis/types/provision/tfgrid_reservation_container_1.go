package provision

import (
	"encoding/json"
	"net"
)

type TfgridReservationContainer1 struct {
	WorkloadId        int64                                 `json:"workload_id"`
	NodeId            int64                                 `json:"node_id"`
	Flist             string                                `json:"flist"`
	HubUrl            string                                `json:"hub_url"`
	Environment       map[string]interface{}                `json:"environment"`
	Entrypoint        string                                `json:"entrypoint"`
	Interactive       bool                                  `json:"interactive"`
	Volumes           []TfgridReservationContainerMount1    `json:"volumes"`
	NetworkConnection []TfgridReservationNetworkConnection1 `json:"network_connection"`
	StatsAggregator   []TfgridReservationStatsaggregator1   `json:"stats_aggregator"`
	FarmerTid         int64                                 `json:"farmer_tid"`
}

func NewTfgridReservationContainer1() (TfgridReservationContainer1, error) {
	const value = "{\"interactive\": true}"
	var object TfgridReservationContainer1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridReservationContainerMount1 struct {
	VolumeId   string `json:"volume_id"`
	Mountpoint string `json:"mountpoint"`
}

func NewTfgridReservationContainerMount1() (TfgridReservationContainerMount1, error) {
	const value = "{}"
	var object TfgridReservationContainerMount1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridReservationNetworkConnection1 struct {
	NetworkId string `json:"network_id"`
	Ipaddress net.IP `json:"ipaddress"`
}

func NewTfgridReservationNetworkConnection1() (TfgridReservationNetworkConnection1, error) {
	const value = "{}"
	var object TfgridReservationNetworkConnection1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
