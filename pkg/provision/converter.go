package provision

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/container/logger"
	generated "github.com/threefoldtech/zos/pkg/gedis/types/provision"
	"github.com/threefoldtech/zos/pkg/network/types"
	"github.com/threefoldtech/zos/tools/bcdb_mock/models/generated/workloads"
)

// ContainerToProvisionType converts TfgridReservationContainer1 to Container
func ContainerToProvisionType(c workloads.TfgridWorkloadsReservationContainer1) (Container, string, error) {
	env := func(m map[string]interface{}) map[string]string {
		o := make(map[string]string)
		for k, v := range m {
			o[k] = fmt.Sprint(v)
		}
		return o
	}
	container := Container{
		FList:        c.Flist,
		FlistStorage: c.HubUrl,
		Env:          env(c.Environment),
		SecretEnv:    env(c.SecretEnvironment),
		Entrypoint:   c.Entrypoint,
		Interactive:  c.Interactive,
		Mounts:       make([]Mount, len(c.Volumes)),
		Logs:         make([]logger.Logs, len(c.Logs)),
	}

	if len(c.NetworkConnection) > 0 {
		container.Network = Network{
			IPs:       []net.IP{c.NetworkConnection[0].Ipaddress},
			NetworkID: pkg.NetID(c.NetworkConnection[0].NetworkId),
		}
	}

	for i, mount := range c.Volumes {
		container.Mounts[i] = Mount{
			VolumeID:   mount.VolumeId,
			Mountpoint: mount.Mountpoint,
		}
	}

	for i, lg := range c.Logs {
		// Only support redis for now
		if lg.Type != logger.RedisType {
			container.Logs[i] = logger.Logs{
				Type: "unknown",
				Data: logger.LogsRedis{
					Stdout: "",
					Stderr: "",
				},
			}
		}

		container.Logs[i] = logger.Logs{
			Type: lg.Type,
			Data: logger.LogsRedis{
				Stdout: lg.Data.Stdout,
				Stderr: lg.Data.Stderr,
			},
		}
	}

	return container, c.NodeID, nil
}

// VolumeToProvisionType converts TfgridReservationVolume1 to Volume
func VolumeToProvisionType(v workloads.TfgridWorkloadsReservationVolume1) (Volume, string, error) {
	volume := Volume{
		Size: uint64(v.Size),
	}
	switch v.Type.String() {
	case "HDD":
		volume.Type = HDDDiskType
	case "SSD":
		volume.Type = SSDDiskType
	default:
		return volume, v.NodeId, fmt.Errorf("disk type %s not supported", v.Type.String())
	}
	return volume, v.NodeId, nil
}

//ZDBToProvisionType converts TfgridReservationZdb1 to ZDB
func ZDBToProvisionType(z workloads.TfgridWorkloadsReservationZdb1) (ZDB, string, error) {
	zdb := ZDB{
		Size:     uint64(z.Size),
		Password: z.Password,
		Public:   z.Public,
	}
	switch z.DiskType.String() {
	case "hdd":
		zdb.DiskType = pkg.HDDDevice
	case "ssd":
		zdb.DiskType = pkg.SSDDevice
	default:
		return zdb, z.NodeId, fmt.Errorf("device type %s not supported", z.DiskType.String())
	}

	switch z.Mode.String() {
	case "seq":
		zdb.Mode = pkg.ZDBModeSeq
	case "user":
		zdb.Mode = pkg.ZDBModeUser
	default:
		return zdb, z.NodeId, fmt.Errorf("0-db mode %s not supported", z.Mode.String())
	}

	return zdb, z.NodeId, nil
}

// K8SToProvisionType converts type to internal provision type
func K8SToProvisionType(k workloads.TfgridWorkloadsReservationK8S1) (Kubernetes, string, error) {
	k8s := Kubernetes{
		Size:          uint8(k.Size),
		NetworkID:     pkg.NetID(k.NetworkId),
		IP:            k.Ipaddress,
		ClusterSecret: k.ClusterSecret,
		MasterIPs:     k.MasterIps,
		SSHKeys:       k.SshKeys,
	}

	return k8s, k.NodeId, nil
}

// NetworkToProvisionType convert TfgridReservationNetwork1 to pkg.Network
func NetworkToProvisionType(n workloads.TfgridWorkloadsReservationNetwork1) (pkg.Network, error) {
	network := pkg.Network{
		Name:         n.Name,
		NetID:        pkg.NetID(n.Name),
		IPRange:      types.NewIPNetFromSchema(n.Iprange),
		NetResources: make([]pkg.NetResource, len(n.NetworkResources)),
	}

	var err error
	for i, nr := range n.NetworkResources {
		network.NetResources[i], err = NetResourceToProvisionType(nr)
		if err != nil {
			return network, err
		}
	}
	return network, nil
}

//WireguardToProvisionType converts WireguardPeer1 to pkg.Peer
func WireguardToProvisionType(p workloads.TfgridWorkloadsWireguardPeer1) (pkg.Peer, error) {
	peer := pkg.Peer{
		WGPublicKey: p.PublicKey,
		Endpoint:    p.Endpoint,
		AllowedIPs:  make([]types.IPNet, len(p.AllowedIprange)),
		Subnet:      types.NewIPNetFromSchema(p.Iprange),
	}

	for i, ip := range p.AllowedIprange {
		peer.AllowedIPs[i] = types.IPNet{ip.IPNet}
	}
	return peer, nil
}

//NetResourceToProvisionType converts TfgridNetworkNetResource1 to pkg.NetResource
func NetResourceToProvisionType(r workloads.TfgridWorkloadsNetworkNetResource1) (pkg.NetResource, error) {
	nr := pkg.NetResource{
		NodeID:       r.NodeId,
		Subnet:       types.NewIPNetFromSchema(r.Iprange),
		WGPrivateKey: r.WireguardPrivateKeyEncrypted,
		WGPublicKey:  r.WireguardPublicKey,
		WGListenPort: uint16(r.WireguardListenPort),
		Peers:        make([]pkg.Peer, len(r.Peers)),
	}

	for i, peer := range r.Peers {
		p, err := WireguardToProvisionType(peer)
		if err != nil {
			return nr, err
		}
		nr.Peers[i] = p
	}

	return nr, nil
}

// WorkloadToProvisionType TfgridReservationWorkload1 to provision.Reservation
func WorkloadToProvisionType(w workloads.TfgridWorkloadsReservationWorkload1) (*Reservation, error) {
	reservation := &Reservation{
		ID:        w.WorkloadId,
		User:      w.User,
		Type:      ReservationType(w.Type.String()),
		Created:   w.Created.Time,
		Duration:  time.Duration(w.Duration) * time.Second,
		Signature: []byte(w.Signature),
		// Data:      w.Content,
		ToDelete: w.ToDelete,
	}

	var (
		data interface{}
		err  error
	)

	switch tmp := w.Content.(type) {
	case workloads.TfgridWorkloadsReservationZdb1:
		data, reservation.NodeID, err = ZDBToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	case workloads.TfgridWorkloadsReservationVolume1:
		data, reservation.NodeID, err = VolumeToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	case workloads.TfgridWorkloadsReservationNetwork1:
		data, err = NetworkToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	case workloads.TfgridWorkloadsReservationContainer1:
		data, reservation.NodeID, err = ContainerToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	case workloads.TfgridWorkloadsReservationK8S1:
		data, reservation.NodeID, err = K8SToProvisionType(tmp)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown workload type (%s) (%T)", w.Type.String(), tmp)
	}

	reservation.Data, err = json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return reservation, nil
}
