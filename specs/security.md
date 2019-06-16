

## Data security
regardless if your connection security clearance, a user can't read data from another container. While this can be implemented at the API level to separate resources attached to certain sub-organization, data encryption also MUST apply so user with physical access to the machine shouldn't be able to unplug a disk and read the data on it.

- ZFS is a very good option since it supports encryption per sub-volume. That means a container that requires data mount can have a dedicated encrypted sub-volume
- An encryption key will be generated per resource, and is stored as part of the resource object (in 3bot, or other orchestrators). The sub-volume lives as long as the container.

## Services capabilities
0> TODO: All daemons on the node must only granted capabilities enough only to do their task, and must be assigned cgroups to limit there memory, cpu, and devices access on the system.