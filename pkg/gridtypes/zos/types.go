package zos

import (
	"encoding/hex"
	"fmt"

	"github.com/threefoldtech/zos/pkg/gridtypes"
)

const (
	// ZMountType type
	ZMountType gridtypes.WorkloadType = "zmount"
	// NetworkType type
	NetworkType gridtypes.WorkloadType = "network"
	// ZDBType type
	ZDBType gridtypes.WorkloadType = "zdb"
	// ZMachineType type
	ZMachineType gridtypes.WorkloadType = "zmachine"
	//PublicIPv4Type type [deprecated]
	PublicIPv4Type gridtypes.WorkloadType = "ipv4"
	//PublicIPType type is the new way to assign public ips
	// to a VM. this has flags (V4, and V6) that has to be set.
	PublicIPType gridtypes.WorkloadType = "ip"
	// GatewayNameProxyType type
	GatewayNameProxyType gridtypes.WorkloadType = "gateway-name-proxy"
	// GatewayFQDNProxyType type
	GatewayFQDNProxyType gridtypes.WorkloadType = "gateway-fqdn-proxy"
	// QuantumSafeFSType type
	QuantumSafeFSType gridtypes.WorkloadType = "qsfs"
	// ZLogsType type
	ZLogsType gridtypes.WorkloadType = "zlogs"
)

func init() {
	// network is a sharable type, which means for a single
	// twin, the network objects can be 'used' from different
	// deployments.
	gridtypes.RegisterSharableType(NetworkType, Network{})
	gridtypes.RegisterType(ZMountType, ZMount{})
	gridtypes.RegisterType(ZDBType, ZDB{})
	gridtypes.RegisterType(ZMachineType, ZMachine{})
	gridtypes.RegisterType(PublicIPv4Type, PublicIP4{})
	gridtypes.RegisterType(PublicIPType, PublicIP{})
	gridtypes.RegisterType(GatewayNameProxyType, GatewayNameProxy{})
	gridtypes.RegisterType(GatewayFQDNProxyType, GatewayFQDNProxy{})
	gridtypes.RegisterType(QuantumSafeFSType, QuantumSafeFS{})
	gridtypes.RegisterType(ZLogsType, ZLogs{})
}

// DeviceType is the actual type of hardware that the storage device runs on,
// i.e. SSD or HDD
type DeviceType string

// Known device types
const (
	SSDDevice DeviceType = "ssd"
	HDDDevice DeviceType = "hdd"
)

func (d DeviceType) String() string {
	return string(d)
}

// Valid validates device type
func (d DeviceType) Valid() error {
	if d != SSDDevice && d != HDDDevice {
		return fmt.Errorf("invalid device type")
	}
	return nil
}

// Bytes value that is represented as hex when serialized to json
type Bytes []byte

// BytesFromHex creates bytes from hex
func BytesFromHex(h string) (Bytes, error) {
	return hex.DecodeString(h)
}

// MustBytesFromHex like BytesFromHex but panics if h is not a valid hex string
func MustBytesFromHex(h string) Bytes {
	bytes, err := BytesFromHex(h)
	if err != nil {
		panic(err)
	}

	return bytes
}

func (h *Bytes) UnmarshalText(text []byte) error {
	data, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}

	*h = data
	return nil
}

func (h Bytes) MarshalText() (text []byte, err error) {
	return []byte(hex.EncodeToString(h)), nil
}
