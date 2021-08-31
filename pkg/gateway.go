package pkg

//go:generate mkdir -p stubs

//go:generate zbusc -module gateway -version 0.0.1 -name manager -package stubs github.com/threefoldtech/zos/pkg+Gateway stubs/gateway_stub.go

type Gateway interface {
	SetNamedProxy(fqdn string, backends []string) error
	DeleteNamedProxy(fqdn string) error
}
