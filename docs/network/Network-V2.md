# 0-OS v2 and it's network

## Introduction

0-OS nodes participating in the Threefold grid, need connectivity of course. They need to be able to communicate over the Internet with each-other in order to do various things:

- download it's OS modules
- perform OS module upgrades
- register itself to the grid, and send regular updates about it's status
- query the grid for tasks to execute
- build and run the Overlay Network
- download flists and the effective files to cache

The nodes themselves can have connectivity in a few different ways:

- Only have RFC1918 private addresses, connected to the Internet through NAT, NO IPv6
  Mostly, these are single-NIC (Network card) machines that can host some workloads through the Overlay Network, but cant't expose services directly. These are HIDDEN nodes, and are mostly booted with an USB stick from bootstrap.grid.tf .
- Dual-stacked: having RFC1918 private IPv4 and public IPv6 , where the IPv6 addresses are received from a home router, but firewalled for outgoing traffic only. These nodes are effectively also HIDDEN
- Nodes with 2 NICs, one that has effectively a NIC connected to a segment that has real public addresses (IPv4 and/or IPv6) and one NIC that is used for booting and local management. (OOB) (like in the drawing for farmer setup)

For Farmers, we need to have Nodes to be reachable over IPv6, so that the nodes can:

- expose services to be proxied into containers/vms
- act as aggregating nodes for Overlay Networks for HIDDEN Nodes

Some Nodes in Farms should also have a publicly reachable IPv4, to make sure that clients that only have IPv4 can effectively reach exposed services.

## Physical setup for farmers

```text
                      XXXXX  XXX
                     XX   XXX  XXXXX XXX
                    X           X      XXX
                   X                     X
                   X      INTERNET       X
                   XXX        X          X
                     XXXXX   XX    XX XXXX
                      +X XXXX  XX XXXXX
                      |
                      |
                      |
                      |
                      |
               +------+--------+
               | FIREWALL/     |
               | ROUTER        |
               +--+----------+-+
                  |          |
      +-----------+----+   +-+--------------+
      |  switch/       |   |  switch/       |
      |  vlan segment  |   |  vlan segment  |
      +-+---------+----+   +---+------------+
        |         |            |
+-------+-------+ |OOB         | PUBLIC
| PXE / dhcp    | |            |
| Ser^er        | |            |
+---------------+ |            |
                  |            |
            +-----+------------+----------+
            |                             |
            |                             +--+
            |                             |  |
            |  NODES                      |  +--+
            +--+--------------------------+  |  |
               |                             |  |
               +--+--------------------------+  |
                  |                             |
                  +-----------------------------+


```

The PXE/dhcp can also be done by the firewall, your mileage may vary.

## Switch and firewall configs

Single switch, multiple switch, it all boils down to the same:

- one port is an access port on an OOB vlan/segment
- one port is connected to a public vlan/segment

The farmer makes sure that every node receives properly an IPv4 address in the OOB segment through means of dhcp, so that with a PXE config or USB, a node can effectively start it's boot process:

- Download kernel and initrd
- Download and mount the system flists so that the 0-OS daemons can start
- Register itself on the grid
- Query the grid for tasks to execute

For the PUBLIC side of the Nodes, there are a few things to consider:

- It's the farmer's job to inform the grid what node gets an IP address, be it IPv4 or IPv4.
- Nodes that don't receive and IPv4 address will connect to the IPv4 net through the NATed OOB network
- A farmer is responsible to provide and IPv6 prefix on at least one segment, and have a Router Advertisement daemon runnig to provide for SLAAC addressin on that segment.
- That IPv6 Prefix on the public segment should not be firewalled, as it's impossible to know in your firewall what ports will get exposed for the proxies.

The Nodes themselves have nothing listening that points into the host OS itself, and are by themselves also firewalled.

## What happens under the hood

## Exposing services that get deployed

## Kuberwhat ? (or VM's in short)
