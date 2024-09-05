package primitives

import (
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	netlight "github.com/threefoldtech/zos/pkg/primitives/network-light"
	"github.com/threefoldtech/zos/pkg/primitives/qsfs"
	vmlight "github.com/threefoldtech/zos/pkg/primitives/vm-light"
	"github.com/threefoldtech/zos/pkg/primitives/volume"
	"github.com/threefoldtech/zos/pkg/primitives/zdb"
	"github.com/threefoldtech/zos/pkg/primitives/zlogs"
	"github.com/threefoldtech/zos/pkg/primitives/zmount"
	"github.com/threefoldtech/zos/pkg/provision"
)

// NewPrimitivesProvisioner creates a new 0-OS provisioner
func NewPrimitivesProvisioner(zbus zbus.Client) provision.Provisioner {
	managers := map[gridtypes.WorkloadType]provision.Manager{
		zos.ZMountType:        zmount.NewManager(zbus),
		zos.ZLogsType:         zlogs.NewManager(zbus),
		zos.QuantumSafeFSType: qsfs.NewManager(zbus),
		zos.ZDBType:           zdb.NewManager(zbus),
		zos.NetworkLightType:  netlight.NewManager(zbus),
		zos.ZMachineLightType: vmlight.NewManager(zbus),
		zos.VolumeType:        volume.NewManager(zbus),
	}

	return provision.NewMapProvisioner(managers)
}
