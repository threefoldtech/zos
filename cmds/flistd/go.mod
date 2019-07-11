module github.com/threefoldtech/zosv2/cmds/flistd

go 1.12

require (
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/rs/zerolog v1.14.3
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/threefoldtech/zbus v0.0.0-20190711111853-91783e8e0ebb
	github.com/threefoldtech/zosv2/modules v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4 // indirect
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7 // indirect
	golang.org/x/sys v0.0.0-20190710143415-6ec70d6a5542 // indirect
	golang.org/x/tools v0.0.0-20190710184609-286818132824 // indirect
)

replace github.com/threefoldtech/zosv2/modules => ../../modules/
