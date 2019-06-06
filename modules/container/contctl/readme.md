# CONTCTL container module debug tool

This small tool allow you to talk to the container module directly.
This is intended to be use for debugging.

Example:

- Start a container

```
 contctl run --flist https://hub.grid.tf/tf-official-apps/threefoldtech-0-db-release-1.0.0.flist --name zdb --entrypoint /bin/zdb
```

- Stop a container:

```
 contctl stop --name zdb
```