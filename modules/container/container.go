package container

import (
	"context"
	"fmt"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/opencontainers/runtime-spec/specs-go"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/namespaces"
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
	root       string
}

func New(root string, zcl zbus.Client, containerd string) modules.ContainerModule {
	if len(containerd) == 0 {
		containerd = containerdSock
	}

	return &containerModule{
		flister:    stubs.NewFlisterStub(zcl),
		containerd: containerd,
		root:       root,
	}
}

func getNetworkSpec(network modules.NetworkInfo) oci.SpecOpts {
	ns := network.Namespace
	if !path.IsAbs(ns) {
		// just name
		ns = path.Join("/var/run/netns", ns)
	}

	return oci.WithLinuxNamespace(
		specs.LinuxNamespace{
			Type: specs.NetworkNamespace,
			Path: ns,
		},
	)
}

func withHooks(hooks specs.Hooks) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, spec *oci.Spec) error {
		spec.Hooks = &hooks
		return nil
	}
}

// NOTE:
// THIS IS A WIP Create action and it's not fully implemented atm
func (c *containerModule) Run(ns string, data modules.Container) (id modules.ContainerID, err error) {
	// create a new client connected to the default socket path for containerd
	client, err := containerd.New(c.containerd)
	if err != nil {
		return id, err
	}
	defer client.Close()

	args, err := shlex.Split(data.Entrypoint)
	if err != nil || len(args) == 0 {
		return id, fmt.Errorf("invalid entrypoint definition '%s'", data.Entrypoint)
	}

	root, err := c.flister.Mount(data.FList, "")
	if err != nil {
		return id, err
	}

	defer func() {
		if err != nil {
			c.flister.Umount(root)
		}
	}()

	opts := []oci.SpecOpts{
		oci.WithDefaultSpecForPlatform("linux/amd64"),
		oci.WithRootFSPath(root),
		oci.WithProcessArgs(args...),
		oci.WithEnv(data.Env),

		// NOTE: the hooks run inside runc namespace
		// it means that we can't do the unmount of the
		// root fs from here.

		// withHooks(specs.Hooks{
		// 	Poststop: []specs.Hook{
		// 		{
		// 			Path: "umount",
		// 			Args: []string{path},
		// 		},
		// 	},
		// }),
	}

	if len(data.Network.Namespace) != 0 {
		opts = append(
			opts,
			getNetworkSpec(data.Network),
		)
	}

	for _, mount := range data.Mounts {
		opts = append(
			opts,
			oci.WithMounts([]specs.Mount{
				{
					Destination: mount.Target,
					Type:        mount.Type,
					Source:      mount.Source,
					Options:     mount.Options,
				},
			}),
		)
	}

	ctx := namespaces.WithNamespace(context.Background(), ns)

	container, err := client.NewContainer(
		ctx,
		data.Name,
		containerd.WithNewSpec(opts...),
	)

	if err != nil {
		return id, err
	}

	defer func() {
		if err != nil {
			container.Delete(ctx, containerd.WithSnapshotCleanup)
		}
	}()

	logs := path.Join(c.root, ns)
	if err = os.MkdirAll(logs, 0755); err != nil {
		return id, err
	}

	task, err := container.NewTask(ctx, cio.LogFile(path.Join(logs, fmt.Sprintf("%s.log", container.ID()))))
	if err != nil {
		return id, err
	}

	// call start on the task to execute the redis server
	if err := task.Start(ctx); err != nil {
		return id, err
	}

	return modules.ContainerID(container.ID()), nil
}

func (c *containerModule) Inspect(ns string, id modules.ContainerID) (modules.ContainerInfo, error) {
	return modules.ContainerInfo{}, nil
}

func (c *containerModule) Delete(ns string, id modules.ContainerID) error {
	client, err := containerd.New(c.containerd)
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := namespaces.WithNamespace(context.Background(), ns)

	container, err := client.LoadContainer(ctx, string(id))
	if err != nil {
		return err
	}

	task, err := container.Task(ctx, nil)
	if err == nil {
		// err == nil, there is a task running inside the container
		exitC, err := task.Wait(ctx)
		if err != nil {
			return err
		}
		trials := 3
	loop:
		for {
			signal := syscall.SIGTERM
			if trials <= 0 {
				signal = syscall.SIGKILL
			}
			task.Kill(ctx, signal)
			trials--
			select {
			case <-exitC:
				break loop
			case <-time.After(1 * time.Second):
			}
		}

		if _, err := task.Delete(ctx); err != nil {
			return err
		}
	}
	spec, err := container.Spec(ctx)
	if err != nil {
		return err
	}

	if err := container.Delete(ctx); err != nil {
		return err
	}

	return c.flister.Umount(spec.Root.Path)
}
