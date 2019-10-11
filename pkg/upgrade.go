package pkg

import (
	"github.com/blang/semver"
)

//go:generate mkdir -p stubs
//go:generate zbusc -module upgrade -version 0.0.1 -name upgrade -package stubs github.com/threefoldtech/zos/pkg+UpgradeModule stubs/upgrade_stub.go

// UpgradeModule zbus interface of the upgrade module
type UpgradeModule interface {
	// version return the current version 0-OS is running
	Version() (semver.Version, error)
}
