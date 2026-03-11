# bins (Runtime Packages)

**File:** `.github/workflows/bins.yaml`
**Trigger:** Push to any branch, or version tags (`v*`)

## Purpose

Orchestrates the build of **all runtime packages** that ZOS depends on. Each package is built in parallel by calling one of the reusable workflows:

- `bin-package.yaml` — standard build + tag (most packages)
- `bin-package-18.04.yaml` — build on Ubuntu 18.04 (for packages requiring older glibc)
- `bin-package-no-tag.yaml` — build without tagging (packages managed independently)

## Packages

### Standard packages (bin-package)

| Package | Description |
|---------|-------------|
| `containerd` | Container runtime |
| `runc` | OCI runtime |
| `virtwhat` | Hypervisor detection |
| `yggdrasil` | Overlay network daemon |
| `rfs` | Remote filesystem |
| `hdparm` | Disk parameters tool |
| `corex` | Container interactive shell |
| `shimlogs` | Container log shim |
| `cloudhypervisor` | VM hypervisor |
| `tailstream` | Log tail streaming |
| `virtiofsd` | Virtio filesystem daemon |
| `vector` | Log/metrics collection |
| `nnc` | Network connectivity checker |
| `lshw` | Hardware lister |
| `cloudconsole` | Cloud console |
| `misc` | Miscellaneous tools |
| `iperf` | Network performance |
| `cpubench` | CPU benchmarking |
| `mycelium` | Mycelium network daemon |

### Ubuntu 18.04 packages (bin-package-18.04)

| Package | Description |
|---------|-------------|
| `mdadm` | RAID management |

### No-tag packages (bin-package-no-tag)

| Package | Description |
|---------|-------------|
| `qsfs` | Quantum Safe File System |
| `traefik` | Reverse proxy |
