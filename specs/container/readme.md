# Container module

This module is responsible to manage containerized applications.
Its only focus is about starting a process with the proper isolation. So there is no notion of storage or networking in this module, this is handled by the layer above.

The container runtime used will be compatible with the [OCI specification](https://github.com/opencontainers/runtime-spec), but which one in particular is still to be decided.

## Interface

```go
type Container struct{
    ID string
    // ...TODO
}

type ContainerConfig struct {
    // Path to the root of the filesystem bundle of the container
    RootFS string
    // Volume to mount bind in the container
    Volume map[string]string
    // Environment variable to set in the container
    Env map[string]string
}

type ContainerModule interface {
    // Create creates a container
    Create(config ContainerConfig) (c Container, err error)
    // Delete deletes any resources held by the container
    Delete(ID string) (error)
    //create and run a container
    Run(config ContainerConfig) (c Container, err error)
    //executes the user defined process in a created container
    Start(ID string) (error)
    // pause suspends all processes inside the container
    Pause(ID string) (error)
    // resumes all processes that have been previously paused
    Resume(ID string) (error)
    // Ps displays the processes running inside a container
    Ps(ID string) (error)
}
```

## OCI bundle

The oci bundle must define `rootfs` and a `config.json`. currently flist defines only the `rootfs` of a container. There are no attached meta to define entrypoint, env variables, exposed ports, etc ...

Since the flist is a tar that container only one file (the rootfs db), it's okay we add more files to the tar. I suggest we add custom meta file that contains
the missing pieces of the bundle that we need to construct the final `config.json` file.

### Flist meta

Part of the `config.json` entries that can be defined byt the user during the flist build stage or when creating a container:

- process.terminal
- process.user
- process.args
- process.env
- process.cwd
- hostname
- mounts (for persistent storage)

Next to these field we will also authorized:

- exposed ports (this will allow auto port forward to the application ports automatically if asked)

### Runtime meta

Parts of the `config.json` that must be chosen by 0-OS during the creating the container on the node:

TODO: The value for each of these field needs to be defined (@maxux, @muhamadazmy, @delandtj), full example for linux: https://github.com/opencontainers/runtime-spec/blob/master/config.md#configuration-schema-example

- process.capabilities
- rlimits
- apparmorProfile
- oomScoreAdj
- selinuxLabel
- noNewPrivileges
- mounts (default one)
- hooks (used for network configuration)
- linux.devices
- linux.sysctl
- cgroupsPath
- resources.network
- resources.pids
- resources.hugepageLimits
- resources.memory
- resources.cpu
- resources.devices
- resources.blockIO
- rootfsPropagation
- seccomp
- namespaces
- maskedPaths
- readonlyPaths
- mountLabel

