package main

import (
	"context"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/threefoldtech/zosv2/modules/flist"
)

func main() {
	if err := caddyExample(); err != nil {
		log.Fatal(err)
	}
}

func caddyExample() error {
	// create a new client connected to the default socket path for containerd
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	// create a new context with an "example" namespace
	ctx := namespaces.WithNamespace(context.Background(), "example")

	// download and mount the caddy flist
	flister := flist.New("")

	path, err := flister.Mount("https://hub.grid.tf/tf-official-apps/caddy.flist", "")
	if err != nil {
		return err
	}
	defer flister.Umount(path)

	// create a container
	container, err := client.NewContainer(
		ctx,
		"caddy-server",
		containerd.WithNewSpec(
			oci.WithDefaultSpecForPlatform("linux/amd64"),
			oci.WithRootFSPath(path),
			oci.WithProcessArgs("caddy")),
	)
	if err != nil {
		return err
	}
	defer container.Delete(ctx, containerd.WithSnapshotCleanup)

	// create a task from the container
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return err
	}
	defer task.Delete(ctx)

	// make sure we wait before calling start
	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		fmt.Println(err)
	}

	// call start on the task to execute the redis server
	if err := task.Start(ctx); err != nil {
		return err
	}

	// sleep for a lil bit to see the logs
	time.Sleep(3 * time.Second)

	// kill the process and get the exit status
	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return err
	}

	// wait for the process to fully exit and print out the exit status
	status := <-exitStatusC
	code, _, err := status.Result()
	if err != nil {
		return err
	}
	fmt.Printf("caddy exited with status: %d\n", code)

	return nil
}
