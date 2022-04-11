# Zmachine

A `Zmachine` is an instance of virtual compute capacity. There are 2 kinds of Zmachines.
One is a `VM`, standard in cloud environments. Next to this it can also be a `container`.
On the Zos level, both of these are implemented as virtual machines. Depending on
the context, it will be considered to be either a VM or a container. In either
scenario, the `Zmachine` is started from an `Flist`.

> Note, both VM and Container on ZOS are actually served as Virtual Machines. The
only difference is that if you are running in VM mode, you only need to provide a **raw**
disk image (image.raw) in your flist.
## VM

In a VM mode, you run your own operating system (for now only linux is supported)
The image provided must be
- EFI bootable
- Cloud-init enabled.

You can find later in this document how to create your own bootable image.

A VM reservations must also have at least 1 volume, as the boot image
will be copied to this volume. The size of the root disk will be the size of this
volume.

The image used to the boot the VM must has cloud-init enabled on boot. Cloud-init
receive its config over the NoCloud source. This takes care of setting up networking, hostname
, root authorized_keys.
### Expected Flist structure

An `Zmachine` will be considered a `VM` if it contains an `/image.raw` file.

`/image.raw` is used as "boot disk". This `/image.raw` is copied to the first attached
volume of the `VM`. Cloud-init will take care of resizing the filesystem on the image
to take the full disk size allocated in the deployment.

Note if the `image.raw` size is larger than the allocated disk. the workload for the VM
will fail.

## Container

A container is meant to host `microservice`. The `microservice` architecture generally
dictates that each service should be run in it's own container (therefore providing
a level of isolation), and communicate with other containers it depends on over the
network.

In Zos, a container is actually also run in a virtualized environment. This is in
contract of the usual approach of running containers on the host. Similar to containers,
some setup is done on behalf of the user. After this is done, the users `entrypoint`
is started. It should be noted that a container has no control over the kernel
used to run it, if this is required, a `VM` should be used instead. Furthermore,
a container should ideally only have 1 process running. A container can be a single
binary, or a complete filesystem. In general, the first should be preferred, and
if you need the latter, it might be an indication that you actually want a `VM`.

For containers, the network setup will be created for you. Your init process can
assume that it will be fully set up (according to the config you provided) by the
time it is started. Mountpoints will also be setup for you. The environment variables
passed will be available inside the container.

### Expected Flist structure

Any Flist will boot as a container, **UNLESS** is has a `/image.raw` file. There is
no need to specify a kernel yourself (it will be provided).

### Known issues
- We need to do proper performance testing for `virtio-fs`. There seems to be some
    suboptimal performance right now.
- It's not currently possible to get container logs.
- TODO: more testing

## Creating VM image
This is a simple tutorial on how to create your own VM image
> Note: Please consider checking the official vm images repo on the hub before building your own
image. this can save you a lot of time (and network traffice) here https://hub.grid.tf/tf-official-vms

### Use one of ubuntu cloud-images
If the ubuntu images in the official repo are not enough, you can simply upload one of the official images as follows

- Visit https://cloud-images.ubuntu.com/
- Select the version you want (let's assume bionic)
- Go to bionic, then click on current
- download the amd64.img file like this one https://cloud-images.ubuntu.com/bionic/current/bionic-server-cloudimg-amd64.img
- This is a `Qcow2` image, this is not supported by zos. So we need to convert this to a raw disk image using the following command
```bash
qemu-img convert -p -f qcow2 -O raw bionic-server-cloudimg-amd64.img image.raw
```
- now we have the raw image (image.raw) time to compress and upload to the hub
```bash
tar -czf ubuntu-18.04-lts.tar.gz image.raw
```
- now visit the hub https://hub.grid.tf/ and login or create your own account, then click on upload my file button
- Select the newly created tar.gz file
- Now you should be able to use this flist to create Zmachine workloads
