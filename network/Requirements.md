## Preamble
-----------
A 0-OS node is defined as the highest granularity of a DC; i.e. every node
implements it's proper reachability for containers/vms that are running in it.

That means that 0-OS needs to be able to set up networking and firewalling for
separated tenant loads.

Also it will be necessary that every node cna then handle it's own IPv6
allocations for the tenant networks.

A tenant load is defined as a service/application that gets exposed over IPv6,
so a tenant network gets a /64 allocation __for each node__ , connected in a
pause container and/or bridge.

For tenant networks to access each other over 0-OS (node) boundaries, a
Wireguard tunnel is built between the tenant bridges, and static routing is
configured.

For each tenant network, there is one exit point, where a tenant could buy an
IPv4 resource if it's available.


## Networking Considerations for 0-OS deployment
-------------

1. ### Nodes deployed in a DC that provides for IPv4 allocations ONLY

That would mean that tenant networks needs to build an 6in4 tunnel to some IPv6
exit point, AND thus have an allocation from the exit point's IPv6 prefix.

On the other hand the IPv4 exit can be handled locally, with a proper IPv4 proxy
into the IPv6 service. So only the exit container/router/gateway needs to be
dual-stack, ever.

2. ### Nodes deployed in a DC that is dual-stack (IPv4/6)

Tunnels 6in4 then do not apply, and IPv4 proxies can be handled locally.


IPv6 allocations are simple to come by, it's just a matter of requesting an
allocation from the LIR (Local Internet Registrar). Mostly they will give a /48
to start with, that gives ample space already. (65536 subnets).  
We then need to be able to hand out an allocation from that range for a tenant
service/network and announce a route, e.g. upstream
`ip route add 2001:fe80:aaaa:1234::/64 via 2001::fe80:aaaa::1234`, which means:
subnet `xxxx/64` is reachable via node `yyyy` (note the placement of the '`::`').

But that kind of administration for aquiring IPv6 allocations need to be done and will require both good documentation of how to obtain them and how to register these resources on the grid to be usable.

Routing:  
In the node's main table that route points to the IP addr of the bridge of that
tenant network (here `2001:fe80:aaaa:1234::`, note: `::` is an ip address, as
there are no 'network numbers' in IPv6).

Firewalling for a destination service will still be ip/port, and IPv4 exposure
can then only be through a proxy for that service.

(There are IPv4->IPv6 nat implementations, but they are flaky at best)

3. ### Nodes deployed in a HOME ipv4-only NAT network

Same as 1. , but where a local IPv4 proxy is not possible, and a 6in4 tunnel to a nearby IPv6 exit, that encompasses an IPv4 service proxy

4. ### Nodes in (more modern) home networks that also have an IPv6 /64 allocation.

Because we need more than one /64 prefix, and that the average home IPv6 router is an outgoing-only firewall, we'll need to to the same as 3. , eventually later do 6in6 for tunneling. All the rest of 3. applies.


# Simplicity

Ultimately, we want to make things as simple as possible, but (sic Albert) not one bit simpler.  
That means that, because of the very existence of the Internet, we can basically piggyback as much as possible on the existing infrastructure, where interconnectivity is already taken care of.


