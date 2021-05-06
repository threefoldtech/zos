module github.com/threefoldtech/zos

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v3 v3.2.2
	github.com/containerd/cgroups v1.0.1
	github.com/containerd/containerd v1.5.0
	github.com/containerd/typeurl v1.0.2
	github.com/containernetworking/cni v0.8.1
	github.com/containernetworking/plugins v0.9.1
	github.com/dave/jennifer v1.4.1 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/firecracker-microvm/firecracker-go-sdk v0.19.1-0.20200110212531-741fc8cb0f2e
	github.com/fsnotify/fsnotify v1.4.9
	github.com/g0rbe/go-chattr v0.0.0-20190906133247-aa435a6a0a37 // indirect
	github.com/gizak/termui/v3 v3.1.0
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-redis/redis v6.15.5+incompatible
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.2.0
	github.com/iancoleman/strcase v0.1.3 // indirect
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6
	github.com/klauspost/compress v1.12.2 // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20200929063507-e6143ca7d51d
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.0
	github.com/rs/zerolog v1.21.0
	github.com/shirou/gopsutil v3.21.4+incompatible
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/termie/go-shutil v0.0.0-20140729215957-bcacb06fecae
	github.com/threefoldtech/0-fs v1.3.1-0.20201203163303-d963de9adea7
	github.com/threefoldtech/tfexplorer v0.5.1-0.20210506083108-5fe51fdf5944
	github.com/threefoldtech/zbus v0.1.4
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0
	github.com/urfave/cli v1.22.5
	github.com/urfave/cli/v2 v2.3.0
	github.com/vishvananda/netlink v1.1.1-0.20201029203352-d40f9887b852
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f
	github.com/whs/nacl-sealed-box v0.0.0-20180930164530-92b9ba845d8d
	github.com/yggdrasil-network/yggdrasil-go v0.3.15-0.20200526002434-ed3bf5ef0736
	go.mongodb.org/mongo-driver v1.5.2 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/crypto v0.0.0-20210505212654-3497b51f5e64
	golang.org/x/net v0.0.0-20210505214959-0714010a04ed
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210503173754-0981d6026fa6
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20191219145116-fa6499c8e75f
	google.golang.org/genproto v0.0.0-20210505142820-a42aa055cf76 // indirect
	google.golang.org/grpc v1.37.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
