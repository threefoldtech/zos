package workloads

import (
	"encoding/json"
	"net"
)

type TfgridWorkloadsReservationK8S1 struct {
	WorkloadId      int64                                        `bson:"workload_id" json:"workload_id"`
	NodeId          string                                       `bson:"node_id" json:"node_id"`
	Size            int64                                        `bson:"size" json:"size"`
	NetworkId       string                                       `bson:"network_id" json:"network_id"`
	Ipaddress       net.IP                                       `bson:"ipaddress" json:"ipaddress"`
	ClusterSecret   string                                       `bson:"cluster_secret" json:"cluster_secret"`
	MasterIps       []net.IP                                     `bson:"master_ips" json:"master_ips"`
	SshKeys         []string                                     `bson:"ssh_keys" json:"ssh_keys"`
	StatsAggregator []TfgridWorkloadsReservationStatsaggregator1 `bson:"stats_aggregator" json:"stats_aggregator"`
	FarmerTid       int64                                        `bson:"farmer_tid" json:"farmer_tid"`
}

func NewTfgridWorkloadsReservationK8S1() (TfgridWorkloadsReservationK8S1, error) {
	const value = "{}"
	var object TfgridWorkloadsReservationK8S1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
