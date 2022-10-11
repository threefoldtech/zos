package pkg

import (
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

// PublicConfigEvent pubic config event received. The type specify if this is just notification
// of the reconnection, or actual event has been received.
type PublicConfigEvent struct {
	PublicConfig substrate.OptionPublicConfig
}

// DeploymentCancelledEvent a contract has been cancelled, The type specify if this is just notification
// of the reconnection, or actual event has been received.
type DeploymentCancelledEvent struct {
	Deployment gridtypes.DeploymentID
	TwinId     uint32
}

// ContractLockedEvent is raised when a contract is locked/unlocked. On locking the Lock flag will be set to true.
// If Kind is EventSubscribed it means event stream has been reconnected and might be events loss. It's up to the
// handler of this event type to make sure contracts are synched with the grid.
type ContractLockedEvent struct {
	Contract gridtypes.ContractID
	TwinId   uint32
	Lock     bool
}

type PowerChangeEvent struct {
	Kind   EventKind
	FarmID FarmID
	NodeID uint32
	Target substrate.PowerTarget
}

