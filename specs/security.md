# Requirements for security

> A resource is a container,  or a VM
- API is only accessible to members of an organization, a JWT must be provided to prof membership
- JWT can has different roles, based on a predefined sub-organizations. (storage, containers, vms, monitor, etc...) *to be defined*
- Any API call that is going to provision a resource that consume cpu, memory, or storage must require a PublicKey.
- The public-key is coupled with the resource for its entire life-time, only requests that are signed with the private-key part are authorized to operate on the resource.
- Data on disk is ALWAYS encrypted, per resource. (A key is generated once storage is required and is )

> Another idea, is that the farmer access only allow him to provision a `space`, a space is a cpu,memory,storage that has it's own API.
The Space API only allow access to the user (owner of the space). The user space API has the endpoints required to provision containers, and or virtual-machines. More or less, a `space` is a reservation of certain capacity on a `single-node`.


## Connection Privilege
The system will never provide a terminal (not in production anyway). The only way to ask for resources from a machine is only via it's API.

The API access, must be protected to prevent un-authorized access or malicious abuse of the node. A connection must gain privilege by providing a JWT token that defines its role.

## Data security
regardless if your connection security clearance, a user can't read data from another container. While this can be implemented at the API level to separate resources attached to certain sub-organization, data encryption also MUST apply so user with physical access to the machine shouldn't be able to unplug a disk and read the data on it.

- ZFS is a very good option since it supports encryption per sub-volume. That means a container that requires data mount can have a dedicated encrypted sub-volume
- An encryption key will be generated per resource, and is stored as part of the resource object (in 3bot, or other orchestrators). The sub-volume lives as long as the container.

## Services capabilities
0> TODO: All daemons on the node must only granted capabilities enough only to do their task, and must be assigned cgroups to limit there memory, cpu, and devices access on the system.