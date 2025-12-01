# VMTestD - VM Test Deployment Service

## Overview

`vmtestd` is a ZOS service that periodically deploys and decommissions test VMs to verify the VM deployment infrastructure is working correctly.

## Features

- **Periodic Testing**: Runs VM deployment tests at configurable intervals (default: 5 minutes)
- **Automatic Cleanup**: Deploys and decommissions VMs automatically
- **Error Reporting**: Logs detailed error messages when deployments fail
- **Zinit Integration**: Runs as a zinit-managed service with proper dependencies

## Architecture

The service follows the same pattern as other ZOS modules like `contd` and `noded`:

1. **Module Entry Point**: Defined in `main.go` with CLI flags
2. **Service Loop**: Runs continuously with periodic test executions
3. **ZBus Integration**: Connects to the message broker to communicate with other modules

## Configuration

### Command Line Flags

- `--root`: Working directory (default: `/var/cache/modules/vmtestd`)
- `--broker`: Message broker connection string (default: `unix:///var/run/redis.sock`)
- `--interval`: Test interval duration (default: `5m`)

### Zinit Service

The service is configured in `/etc/zinit/vmtestd.yaml`:

```yaml
exec: vmtestd --broker unix:///var/run/redis.sock --root /var/cache/modules/vmtestd --interval 5m
after:
  - boot
  - vmd
  - networkd
```

## Dependencies

The service depends on:
- `boot`: System initialization
- `vmd`: VM module daemon
- `networkd`: Network daemon

## Implementation Notes

The service performs the following operations:

1. **Connects to VM module** via ZBus
2. **Lists existing VMs** to verify connectivity
3. **Mounts the flist** using the flist module
4. **Inspects flist contents** to determine:
   - If it's a container (no `/image.raw`) or full VM
   - Kernel path (`/boot/vmlinuz`)
   - Initrd path (`/boot/initrd.img`)
   - Disk image path (`/image.raw`)
5. **Creates VM configuration** with:
   - Name: `test-vm-<timestamp>`
   - CPU: 1 core
   - Memory: 512 MB
   - Flist: `https://hub.threefold.me/tf-official-apps/alpine3.flist`
   - Boot type: BootVirtioFS (container) or BootDisk (full VM)
6. **Deploys the VM** using `vmd.Run(ctx, vmConfig)`
7. **Waits 10 seconds** for the VM to run
8. **Decommissions the VM** using `vmd.Delete(ctx, vmID)`
9. **Unmounts the flist** to clean up

The service logs all operations and errors for monitoring purposes.

## Building

The module is automatically included when building the main `zos` binary:

```bash
go build -o zos ./cmds/zos
```

## Running

### As a standalone service:
```bash
vmtestd --broker unix:///var/run/redis.sock --interval 1m
```

### Via zinit:
```bash
zinit start vmtestd
zinit status vmtestd
zinit log vmtestd
```

## Monitoring

Check the service logs:
```bash
zinit log vmtestd
```

## Future Enhancements

- Add metrics collection for deployment success/failure rates
- Implement actual VM deployment with configurable VM specifications
- Add health check endpoints
- Support for different VM test scenarios (network, storage, etc.)
- Integration with monitoring/alerting systems
