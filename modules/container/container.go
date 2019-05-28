package container

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/google/shlex"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zosv2/modules"
	"github.com/threefoldtech/zosv2/modules/stubs"
)

const (
	containerdSock = "/run/containerd/containerd.sock"
)

type containerModule struct {
	flister    *stubs.FlisterStub
	containerd string
}

func New(zcl zbus.Client, containerd string) modules.ContainerModule {
	if len(containerd) == 0 {
		containerd = containerdSock
	}

	return &containerModule{
		flister:    stubs.NewFlisterStub(zcl),
		containerd: containerd,
	}
}

// NOTE:
// THIS IS A WIP Create action and it's not fully implemented atm
func (c *containerModule) Run(name string, flist string, tags []string, network modules.NetworkInfo,
	mounts []modules.MountInfo, entrypoint string) (id modules.ContainerID, err error) {
	// create a new client connected to the default socket path for containerd
	client, err := containerd.New(c.containerd)
	if err != nil {
		return id, err
	}
	defer client.Close()

	args, _ := shlex.Split(entrypoint)

	// create a new context with an "example" namespace
	// ctx := namespaces.WithNamespace(context.Background(), ns)
	ctx := context.Background()
	path, err := c.flister.Mount(flist, "")
	if err != nil {
		return id, err
	}

	defer func() {
		if err != nil {
			c.flister.Umount(path)
		}
	}()

	// create a container
	container, err := client.NewContainer(
		ctx,
		name,
		containerd.WithNewSpec(
			oci.WithDefaultSpecForPlatform("linux/amd64"),
			oci.WithRootFSPath(path),
			oci.WithProcessArgs(args...)),
	)

	if err != nil {
		return id, err
	}

	defer func() {
		if err != nil {
			container.Delete(ctx, containerd.WithSnapshotCleanup)
		}
	}()

	// create a task from the container
	// TODO: change the creator to use a redirected output to a log
	// file instead.
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return id, err
	}

	// call start on the task to execute the redis server
	if err := task.Start(ctx); err != nil {
		return id, err
	}

	return id, nil
}

func (c *containerModule) Inspect(id modules.ContainerID) (modules.ContainerInfo, error) {
	return modules.ContainerInfo{}, nil
}

func (c *containerModule) Delete(id modules.ContainerID) error {
	return nil
}
