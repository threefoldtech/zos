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

But we need to stress the importance of IPv6 availability when you're running a multi-node farm in a datacentre: as the grid is boldly claiming to be a new Internet, we should make sure we adhere to the new protocols that are future-proof. Hence: IPv6 is the base, and IPv4 is just there to accomodate the transition.

Nowadays, RIPE can't even hand out consecutive /22 IPv4 blocks any more for new LIRs, so you'll be bound to market to get IPv4, mostly at rates of 10-15 Euro per IP. Things tend to get costly that way.

So anyway, IPv6 is not an afterthought in 0-OS, we're starting with it.

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

The Nodes themselves have nothing listening that points into the host OS itself, and are by themselves also firewalled. In dev mode, there is an ssh server with a key-only login, accessible by a select few ;-)

## DHCP/Radvd/RA/DHCP6

For home networks, there is not much to do, a Node will get an IPv4 Privete(rfc1918) address , and most probaly and ipv6 address in a /64 prefix, but is not reachable over ipv6, unless the firewall is disabled for IPv6. As we can't rely on the fact that that is possible, we assume these nodes to be HIDDEN.

A normal self-respecting Firewall or IP-capable switch can hand out IP[46] addresses, some can even bootp/tftp to get nodes booted over the network.
We are (full of hope) assuming that you would have such a beast to configure and splice your network in multiple segments. A segment is a physical network separation. That can be port-based vlans, or even separate switches, whatver rocks your boat, the keyword is here **separate**.

On both segments you will need a way to hand out IPv4 addresses based on MAC addresses of the nodes. Yes, there is some administration to do, but it's a one-off, and really necessary, because you really need to know whic physical machine has which IP. For lights-out management and location of machines that is a must.

So you'll need a list of mac addresses

## What happens under the hood (farmer)

While we did our uttermost best to keep IPv4 address needs to a strict minimum, at least one Node will need an IPv4 address for handling everything that is Overlay Networks.
For Containers to reach the Internet, any type of connectivity will do, be it NAT or though an Internal DMZ that has a routable IPv4 address.

Internally, a lot of things are being set-up to nave a node porperly participate in the grid, as well to be prepared to partake in the User's Overlay Networks.

A little Drawing :

For now, we have a typical Node in a farm to behave thusly:

```text

```

## Exposing services that get deployed

## Kuberwhat ? (or VM's in short)
