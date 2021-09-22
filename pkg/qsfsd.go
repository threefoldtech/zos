package pkg

import "github.com/threefoldtech/zos/pkg/gridtypes/zos"

//go:generate mkdir -p stubs

//go:generate zbusc -module qsfsd -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+QSFSD stubs/qsfsd_stub.go

type QSFSD interface {
	Mount(wlID string, cfg zos.QuatumSafeFS) (string, error)
	Unmount(wlID string) error
}
