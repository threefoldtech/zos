## Routing in Circular Meshes with Wireguard

As we are leveraging WireGuard as building block for communications that are inter-node **per** user network, each Network Resource receives it's own IPv6 Prefix, and eventually it's own RFC1918 private IPv4 subnet.

Either way we want IPv6 to function, there will be a need for that circular network to be able to ultimately have a route to the BBI (Big Bad Internet). So every User Network will have an NR that is also an exit, with the appropriate default gateway, being an upstream route.

_That also means that the Upstream Router needs to know the route back to the Prefixes/subnets that live in that circular net._ (more on that later)

### Network Resource Containers

In the Network Resource Container, a single WireGuard interface is set-up with every network resource of that user network as peer, with `AllowedIPs` containing the Link-Local IPv6 address and Prefix of that Peer.
These prefixes are exactly reflected as routes in the Network Resource Container, in time we could als envision adding IPv4, where bits 8-23 are deterministically derived from the Prefix's last 2 nibbles (a nibble is one byte in an IPv6 address).

For example :

  - Considering we received an IPv6 Allocation of : `2a02:1802:5e::/48`
    - that gives us possibility for `64 - 48 = 16` -> `2^16 = 65536` possible network resource containers, each having a `/64` Prefix

  - A valid Prefix for an NR is: `2a02:1802:5e:010a::/64`
    - having `2a02:1802:5e:10a::1/64`
    - and having : `10.1.10.254/24` (`01 = 1 and 0a = 10`)
    - for IPv4 routing to work, the WireGuard interface will also need an IPv4 address (`172.16.1.10/16`)

  - Every NR Container has a route for all Peer Prefixes/subnets.

### Exit Containers

The Exit Container is basically an NR with an added interface and Link-Local IPv6 address that is connected to the Upstream network segment, be it IPv6 or IPv4.  
For IPv6, in terms of routing, it has only a default gateway, so it can send packets on their merry way for all routes its has no knowledge of.  
TBD : will this be a separate container, connected to a bridge that itself is also connected to an NR... 
  - So the exit network interface (a veth pair, an OVS port, a physical interface (like an SRIOv VF)):
  ```

  ```

### Upstream Routers / L3-Switches

In order for manually set-up routes to work properly, both ends of each peer need to know their routes.  
  - For an exit container, that's easy: a defalt route.
  - For the Upstream router (the machine/switch/router having the ultimate default route), each allocated prefix to an NR needs to be forwarde to the exit container of the Network (that Circular Mesh) containing the NR for packets to be able to return.


### IP Address Allocations for containers

### IP Address Allocations for VMs

