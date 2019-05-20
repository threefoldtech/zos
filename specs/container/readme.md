# Container module

This module is responsible to manage containerized applications.
Its only focus is about starting a process with the proper isolation. So there is no notion of storage or networking in this module, this is handled by the layer above.

The container runtime used will be compatible with the [OCI specification](https://github.com/opencontainers/runtime-spec), but which one in particular is still to be decided.

## Design

One of the requirements of the container module is that any container running on 0-OS should not be affected by an upgrade/restart of any 0-OS module.
In order to do that, we need to have shim process that keeps the file descriptor of the container open in case of a restart of the container module. The shim process is going to be the one responsible to talk to the OCI runtime to manage the container itself.

Next is a simplify version of the lifetime flow a container:

![flow](../../assets/Container_module_flow.png)

## Implementation

Most of the work to implement such a system as already been done by other. Mainly containerd has some very nice libraries around
shim and runc.

Here is a list of link of interest:
- [containerd](https://github.com/containerd/containerd)
  - specifically the runtime package: https://github.com/containerd/containerd/tree/master/runtime
- [containerd client example](https://github.com/containerd/containerd/blob/master/docs/getting-started.md)
- [list of project that have integrated containerd](https://github.com/containerd/containerd/blob/master/ADOPTERS.md)
- [firecracker shim design](https://github.com/firecracker-microvm/firecracker-containerd/blob/master/docs/shim-design.md)