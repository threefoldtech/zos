package workloads

import (
	"encoding/json"
	"net"
)

type TfgridWorkloadsReservationContainer1 struct {
	WorkloadId        int64                                          `bson:"workload_id" json:"workload_id"`
	NodeId            string                                         `bson:"node_id" json:"node_id"`
	Flist             string                                         `bson:"flist" json:"flist"`
	HubUrl            string                                         `bson:"hub_url" json:"hub_url"`
	Environment       map[string]interface{}                         `bson:"environment" json:"environment"`
	SecretEnvironment map[string]interface{}                         `bson:"secret_environment" json:"secret_environment"`
	Entrypoint        string                                         `bson:"entrypoint" json:"entrypoint"`
	Interactive       bool                                           `bson:"interactive" json:"interactive"`
	Volumes           []TfgridWorkloadsReservationContainerMount1    `bson:"volumes" json:"volumes"`
	NetworkConnection []TfgridWorkloadsReservationNetworkConnection1 `bson:"network_connection" json:"network_connection"`
	StatsAggregator   []TfgridWorkloadsReservationStatsaggregator1   `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid         int64                                          `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationContainer1() (TfgridWorkloadsReservationContainer1, error) {
	const value = "{\"interactive\": true}"
	var object TfgridWorkloadsReservationContainer1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationContainerMount1 struct {
	VolumeId   string `bson:"volume_id" json:"volume_id"`
	Mountpoint string `bson:"mountpoint" json:"mountpoint"`
}

func NewTfgridWorkloadsReservationContainerMount1() (TfgridWorkloadsReservationContainerMount1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationContainerMount1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridWorkloadsReservationNetworkConnection1 struct {
	NetworkId string `bson:"network_id" json:"network_id"`
	Ipaddress net.IP `bson:"ipaddress" json:"ipaddress"`
}

func NewTfgridWorkloadsReservationNetworkConnection1() (TfgridWorkloadsReservationNetworkConnection1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationNetworkConnection1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
