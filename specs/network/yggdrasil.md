# Yggdrasil integration in 0-OS

Since day one, 0-OS v2 networking has been design around IPv6. The goal was avoid having to deal with exhausted IPV4 address and be ready for the future.

While this decision made sense on the long term, it pose trouble on the short term for farmer that only have access to ipv4  and are unable to ask for an upgrade to their IPS.

To workaround this problem, the idea is to have enough node in the grid with dual stack and leverage some of them to create some tunnels encapsulating ipv6 traffic. The ultimate goal being to ipv6 enable the nodes that needs it.

## Yggdrasil

[Yggdrasil network project](https://yggdrasil-network.github.io/) has been selected to be used to manages these tunnels.

## Integration in 0-OS

Yggdrasil daemon will be started automatically on all nodes with multicast discovery enable on zos interface. The goal is all nodes in a farm discover themself. For the node with a public config network namespace, yggdrasil needs to have its listening port in the public namespace.

Farmer will then be able to configure some of their node to become tunnels providers. This means the farmer will allocate a /64 ipv6 prefix that can be used by ipv4 only nodes to create those tunnels.

One the side of ipv4 only nodes. When booting, networkd will detect there is no ipv6 available. It will then try to find some tunnel provider nodes and auto configure itself. TODO: define this flow in more detail.
Once the tunnel is configured, any time the node needs to get an ipv6 address for its workloads, be it for 0-DB or user containers, it will take it from the allocated ipv6 prefix. The traffic will then be forwarded to the nodes providing the tunnel.

Some more advanced topology for big farm could be also created where a single node in the farm provides ipv6 access for the rest of the node of the farm.