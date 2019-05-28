# Run 0-OS in a VM using qemu

This folder contains a script that you can use to run 0-OS in a VM using qemu.

For ease of use a Makefile is also provided. To prepare your environment call `make prepare`. This will :

- copy `zinit` binary into the overlay
- download a 0-OS kernel

You also need to create a bridge called `zos0` and have a dhcp server giving out IP on the bridge range.

Example bridge config:

```
Description="zos0"
Interface=zos0
Connection=bridge
IP=static
Address=172.20.0.1/24 # use a range that fits your network
DNS=(8.8.8.8)
```

Example dnsmasq config:

```
domain=172.20.0.0/24,local
interface=lxc0
dhcp-range=172.20.0.10,172.20.0.100,24h
```

To start the 0-OS VM, do `make start`