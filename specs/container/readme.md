# Container module

This module is responsible to manage containerized applications.
Its only focus is about starting a process with the proper isolation. So there is no notion of storage or networking in this module, this is handled by the layer above.

The container runtime used will be compatible with the [OCI specification](https://github.com/opencontainers/runtime-spec), but which one in particular is still to be decided.

### Interface

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

# OCI bundle
The oci bundle must define `rootfs` and a `config.json`. currently flist defines only the `rootfs` of a container. There are no attached meta to define entrypoint, env variables, exposed ports, etc ...

Since the flist is a tar that container only one file (the rootfs db), it's okay we add more files to the tar. I suggest we add custom meta file that containes
the messing pieces of the bundle that we need to construct the final `config.json` file.

## Flist meta
List of config.json entries that can be defined during the flist build stage.
- args (this defines entrypoing exec and full arguments list)
- env (environ variable to be available inside the contianer)
- exposed ports (this will allow auto port forward to the application ports automatically if asked)
- available mount destinations (?)

## Runtime meta
Parts of the `config.json` that must be chosen during the `runtime` or the time of creating the container on the node
- Privilege profile. We will probably have few pre-defined profiles for privilege (privileged and non-privileged).
  - cgroups (cpu, memory, devices, etc...)
  - capabilities (libcap)
- Mounts.
- Hostname.
- Network namespaces and configurations.

# Profiles
We supported only 2 profiles with zos v1 as follows:

## Unprivileged
- No access to node devices (device cgrpups)
- Reduced capabilities (libcap)
- CPU and Memory cgroups
- Isolated users, pids, ns, and uts namespaces

TODO: define the base `config.json` that limit the container to these limitations

## Privileged
- Full access to device nodes
- Full capabilities
- No CPU, Memory limits
- Isolated users, pids, ns, and uts namespaces

TODO: define the base `config.json` that applies these limitations

## Host networking
This is not a separate profile, but actually can be applied with both Privileged and Unprivileged profiles.
Once the host network is set, no new `ns` (network namespace) is created so containers shares the same networks
stack with the node.