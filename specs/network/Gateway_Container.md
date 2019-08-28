# Routing in ExitNodes

## Upstream Routers, ExitNodes, Gateway, IPv4 Nat

To get packets routed, you need to tell what goes where on every point that shuffles packets back and forth between interfaces.  
For everything that we control ourselves, things are simple, we know what goes where.  
It's more difficult to know for the upstream routers, where the Network Resources are located, i.e. behind what Exitpoint in what ExitNode.  
Keeping in mind that a `/48` allocation is 65536 IPv6 Prefixes (Subnets), it will be difficult for an upstream router to know all that, the more we receive a pseudo-random number for a prefix. Also, a lot of upstream routers (ASIC based) are totally incapable of handling 2^16 static routes. Another caveat is that although upstream routers can do bgp/ospf, they're not designed for such big routing tables, unless they're big Internet Backbone routers.
On the upstream part, we need it to be simple, no doubt about that, as having to run and maintain a routing protocol is going to be an unsurmountable problem for your Mom & Pop Farmer.

How we're using IPv6 (and IPv4) is not like 'normal' IPv6 planning goes in terms of slicing and dicing per department/locality/service...
Routes are per `/64` prefix and we need to be able to have them all. Also, 2^16 routes are.. erm.. not common, to say the least.

So how do we handle that?

First, on the upstream router, we need to make sure that as little as possible needs to be done and that it is already setup for scaling to bigger environments. Hence: as few routes as possible, and as simple to setup as possible.

Secondly, for now routing protocols __internally__ in the grid will generate more problems than they fix, because the dynamics in a grid will definitely put a strain on these routing protocols.

The requirements for a 'Farm' that provides for Tenant Network Exitpoints is that the 'Farmer' has an IPv6 allocation from their ISP and/or RIR for a `/48` network.
That gives us `65536` subnets to hand out to Network Resources. But that is a BIG number for static routes, so we need to split that in more manageable chuncks, luckily CIDR is made for that.
A 'decent' linux system can handle a few 1000 routes, but we haven't seen what the upper limit would be. So let's split that `/48` in 16 equal parts, where the scaling can then be handeled by adding ExitNodes to handle routes.
So these 16 `/52` networks allow for 15 ExitNodes, the first part of the 16 being reserved for internal purposes.

Every ExitNode's Gateway Container handles the routing for the ExitPoints in the ExitNode.
The gateway container is the routing endpoint (from the Upstream's point) for that 8th part. That way, in the Upstream router, there are only max 16 static routes.
Basically that allows for 4096 routes in the Gateway, that might aver to be very big in the long run, but for now we won't have to worry about that too much... Or maybe we do, we'll advise after the Alpha release.

### Network requirements and setup

There are requirements, of course.

- You have an IPv6 allocation and it's routed and announced.
- You have a small IPv4 allocation that is routable (not a single IP)
- ExitNodes need to have some decent network cards (Intel (I have Intel stock ;-) ))
- Your upstream Router can do IPv6 (really)
  - with PrefixZero configured and routable
  - with some of the 16 static routes added
- Your ExitNodes have at least 2 network cards.

#### Setting up Routes towards the ExitNode's Gateway Containers

Every router/l3 switch/firewall will have their specific ways to configure routing, but basically you'll need to add a route for every ExitNode you enabled, where the routes will always be deterministic, with only the allocation prefix as parameter. That makes things as simple as possible.

#### Gateway Containers

For the gateway containers to function properly, we can better have separate NICs in order to separate the effective bandwith for exit node execution (like 0-DB, and other Public-exitpoint IPv6 traffic) from Forwarding itself.
As routing in linux is not very cpu-hungry (provided you have decent NICs), we can easily run our router in the nodes themselves.

The other big advantage is that we can easily recreate the Gateway container and attached Exitpoints in another Node if one of the Exit nodes fails.

In a later phase it would be even (easily) possible to have the Gateway Containers in an active-active setup with VRRP or with some BGP or other routing protocol so that failure of an ExitNode doesn't bring down all the network resources attached to it, as there can be many (max 4k, but that is a lot).

#### PrefixZero

PrefixZero is the 0th network of an allocation but for us it's als extended to 1/16th of that allocation, so for the Prefixes of the NRs, we start at `$prefix:1000::/52` to `$prefix:f000::/52`, where every Exitnode handles the Exitpoints in that 'subnet'.

#### Opinionated, simplistic, deterministic, your pick

#### Implementation details

```ascii
         +-+ to [::]/0
         |      0.0.0.0/0
         |
+--------+---------------------------------------------------------------+
|                                                                        |
|                      UPSTREAM ROUTER                                   |
|    has                  routes                                         |
|    $prefixzero::1/64    $prefix:1::/52 via fe80::1:0:0:0:1 dev swp1    |
|           fe80::1/64              or via $prefix:1:0:0:0:1             |
+------+-----------------------------------------------------------------+
       |
       |                                           PrefixZero
+------+---+--------------------------------------------------------+
           |
           |
           |                                   ExitNode1
+------------------------------------------------------------+
|          |         GatewayContainer                        |
| +--------------------------------------------------------+ |
| |        +                                               | |
| |   iface pub-1-1                                        | | GWPubName   (Gateway Routing iface)
| |      fe80::1:0:0:0:1/64                                | | GWPubLL     (Gateway Route LL )
| |    $prefix:1:0:0:0:1/64                                | | GWPubIP6    (Gateway Route IPv6)
| |    185.69.166.123/24                                   | | GWPubIP4    (Gateway IPv4 for -> SNAT only)
| |                                                        | |
| | iface to-1abc-1               iface to-1123-1          | | GWtoEPName  (ExitPoint's veth peer name)
| |    fe80::1:1abc/64               fe80::1:1123/64       | | GWtoEPLL    (Gateway to ExitPoint Link-Local)
| |    10.1.0.1/32                   10.1.0.1/32           | |
| |         +                            +                 | |
| +--------------------------------------------------------+ |
|           |                            |                   |
| +-----------------------+    +-----------------------+     |
| |         +             |    |         +             |     |
| |  iface  pub-1abc-1    |    |  iface  pub-1123-1    |     | EPPubName   (ExitPoint Pub iface name)
| |   fe80::1abc:1/64     |    |   fe80::1123:1/64     |     | EPPubLL     (ExitPoint Pub Link-Local)
| |  10.255.26.188/32     |    |  10.255.17.35/32      |     | EPPubIP4R   (ExitPoint Pub routing IPv4 on lo)
| | Network Resource      |    | Network Resource      |     |
| | - iface wg-1abc-1     |    | - iface wg-1123-1     |     | WGName      (Wireguard iface name)
| |   fe80::1abc/64       |    |   fe80::1123/64       |     | WGLL        (Wireguard Link-Local addr)
| |   10.255.26.188/24    |    |   10.255.17.35/24     |     | WGIP4RT     (Wireguard IPv4 routing address)
| | - iface veth-1abc-1   |    | - iface veth-1123-1   |     | NRLocalName (NR Local interface name )
| |   pref:1abc::1/64     |    |   pref:1123::1/64     |     | NRLocalIP6  (NR local IPv6 addr)
| |   10.26.188.1/24      |    |   10.17.35.1/24       |     | NRLocalIP4  (NR local IPv4 addr)
| |                       |    |                       |     |
| +-----------------------+    +-----------------------+     |
|   ExitPoint                     ExitPoint                  |
+------------------------------------------------------------+
                                               ExitNode1
 routes in Gateway (BAR)
 default via fe80::1 dev pub-1-1 # GWPubName(ExitnodeNR,AllocationNr)
 $prefix:1abc::/64 via fe80::1abc:1 dev to-1abc-1
 $prefix:1123::/64 via fe80::1123:1 dev to-1123-1

 routes in Exitpoint (only penultimate, the rest is handeled by the NR code)
 default via fe80::1000:1abc dev pub-1abc-1
```

Prefixes per ExitNode

```text
ExitNode1 -> 1000
ExitNode2 -> 2000
ExitNode3 -> 3000
ExitNode4 -> 4000
...
ExitNode10 -> a000
...
ExitNode14 -> e000
ExitNode15 -> f000
```

I hope the drawing speaks for itself, but in layman's terms, each ExitNode needs to become a BAR (big ass router). As BAR, it will be responsible for a 16th part of the Allocation a farmer has received.

We're speaking __always__ about a `/48`... Period. For other Allocations we'll explain afterwards in more elaborated documentation.

That basically means :

An Allocation prefix has 6 bytes:
`2a02:1802:005e:xxxx:yyyy:yyyy:yyyy:yyyy` where the numbers are fixed and routable to our router, and the `x & y` is what we get to play with.

Thus the 1st 4 `x` is the number of subnets in `/64` we can route. A `/64` 'subnet' is a big (eufemism) address space, we admit, but it is the smallest unity in terms of physical segments in the IPv6 world.
So we get 2 bytes to address `/64` networks... That means 65536 Network Resources that we can route with that Allocation.
In router terms, that are **a lot** of routes, so we need to split that up. Even as BAR, 4K routes are many, but in the realms of feasible, so let's go for that.
One caveat, though, is that every time you get +2K routes, it is advisable to add an ExitNode in that Allocation. That is not 'bad' per se, but you will have to make sure that Node is properly set-up physically that it can act as one. In a standard Farm setup, we would even advise that every node in that farm is setup alike, so that any node can partake in these routing schemes, and thus that every node can act as backup for eventual failure of nodes that act as ExitNode.

So for any Network Resource in a Tenant Network the default is max 2 routing hops away, while the fanout is kept to a limited range.

Actions :

- When a __new__ Tenant network is created, __and__ the Node is an ExitNode, that is the place where the ExitPoint is created.
That means :
  - A Public namespace is created for instantiating the Wireguard interfaces that get sent into the ExitPoint
  - A Gateway Namespace (GW) is created (if it doesn't already exist) and the first route to that TN is added (the prefix of the Network Resource of that ExitPoint)
- The ExitPoint has a veth pair with one end into the Exitpoint and the other in the GW.
- Every time a TN is updated having a new NR, the GW updates it's routing table so that the new prefixpoints to the ExitPoint (which handles the routing for the TNs)
- In a later stage we'll define the GW as an NS with route-leaking VRFs per TN, so that we can forego having to maintain a routing/forwarding filter for the TNs.

TODO: figure out how we can update routing tables when NRs get removed from a TN as easily as possible (provisioning is easy, things get a lot more hairy when removing items)

TODO: we need to make this one level higher in terms of hierarchy: a (non-redundant for now) ExitNode can only get allocations from the TNoDB (whatever it's incantation) for which it is the GW. That means that when handing out a prefix for a new Network Resource, the TNoDB needs to take in account in what 16th it needs to allocate from.

On Node level, we know what to do:

- Updating a TNo for an Exitpoint, also updates the routing table in the BAR of the ExitNode.
- Every ExitNode is responsible for one 16th of the `/48` address space.
- The BAR does the IPv4 firewalling and NATing for the TN.
  
TODO:

- in terms of scalability for an ExitPoint, 3 byte addressing (4096) still seems a lot, but I don't think it would be a big issue, as arriving there would mean that a farm has 65536 possible network resources running, each with the possibility of 256 (IPv4) services in every Node where that TNo lives. That would be Amazon-big. But nibbling at bits on the TNoDB side would be sufficient to still adhere to our deterministic naming/allocations, so we can easily fix that.
- Wireguard being our first overlay, there are other overlays that can be more efficient, like vxlan.
- Sometimes overlay networks can be overkill and costly when we need to optimize for bandwidth, where dedicated NICs and vlans can make things static and still performant.
