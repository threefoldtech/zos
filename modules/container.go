package modules

//go:generate mkdir -p stubs
//go:generate zbusc -module container -version 0.0.1 -name container -package stubs github.com/threefoldtech/zosv2/modules+ContainerModule stubs/container_stub.go

type ContainerID string

type NetworkInfo struct {
	// Currently a container can only join one (and only one)
	// network namespace that has to be pre defined on the node
	// for the container tenant

	// Containers don't need to know about anything about bridges,
	// IPs, wireguards since this is all is only known by the network
	// resource which is out of the scope of this module
	Namespace string
}

type MountInfo struct {
	Source string // source of the mount point on the host
	Target string // target of mount inside the container
}

type ContainerInfo struct {
	ID ContainerID
	//Container info
	Name    string
	Flist   string
	Tags    []string
	Network NetworkInfo
	Mounts  []MountInfo

	// NOTE:
	// Port forwards are not defined by the container. It can be defined
	// by the Network namespace resource. BUT ideally, no port forwards
	// will ever be needed since all is gonna be routing based.
}

type ContainerModule interface {
	// Run creates and starts a container on the node. It also auto
	// starts command defined by `entrypoint` inside the container
	// ns: tenant namespace
	// name: name of container
	// flist: flist of container
	Run(ns, name, flist string, tags []string, network NetworkInfo,
		mounts []MountInfo, entrypoint string) (ContainerID, error)

	// Inspect, return information about the container, given its container id
	Inspect(id ContainerID) (ContainerInfo, error)
	Delete(id ContainerID) error
}
