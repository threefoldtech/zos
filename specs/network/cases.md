
### A Network ID defines interconnected networks and their (single?) exit.

```
type network {
    // some type of ientification... an uuid ?
    // that netid is bound to a user and an allowed (bought) creation of a
    // node-local prefix for a bridge/container/vm
    // needs to be queried from somewhere(TBD) to be filled in
    netid
    // a netresource is a group of interconnected prefixes for a netid
    // needs to be queried and updated when the netresource is created
    []netresource 
    // the exit is the ultimate default gateway container
    // as well the prefix as the local config needs to be queried. 
    // - the prefix from the grid
    // - the exit prefix and default gw from the local allocation
    exit
}

type netresource {
    // where does it live
    nodeid
    // prefix is the IPv6 allocation that will be connected to the 
    // bridge/container/vm
    prefix
    // what are the peers:
    // each netresource needs to know what prefixes are reachable through
    // what endpoint. Basically this `connected` array will be used to build 
    // up the wireguard config in each netresource.
    []connected
}

// the exit container(ns) hold as well a prefix as netresource as well
// an IPv6 address that is going to hold the routes for all prefixes of the
// network. That IPv6 address will thus be the gateway for all prefixes that
// are part of that network. That also means that an upstream router needs to
// know the list of prefixes that need to be routed to that IPv6 address.
// An upstream router is the entry point toward nodes that have only IPv6 acces
// throug tunnels (like nodes in ipv4-only networks or home networks)
type exit {
    // netresource is the same as on all other netresources of a tenant network
    netresource
    // the ultimate IPv{4,6} config of the exit container.
    ipv4_conf
    ipv6_conf
}

type ipv4_conf {
    // cidr
    ipaddress 
    gateway
    metric
    // deterministic name in function of the prefix and it's allocation
    interface
    // TBD, we need to establish if we want fc00/7 (ULA) or rfc1918 networks 
    // to be NATed (6t04 and/or 66)
    nat
}
// comments: see above
type ipv6_conf {
    ipaddress // cidr
    gateway
    metric
    interface
}

// connected are the peers for which we have a tunnel establised, and the
// prefix it routes to. The connection, as it is a peer to peer connection,
// can be of type wireguard, but also any other type that can bring
// a packet to a node containing a netresource.
// If for instance that node lives in the same subnet, it'll be a lot more
// efficient to set up a vxlan (multicast or with direct fdb entries), than
// using wireguard tunnels (that can be seen in a later phase)
type connected {
    connection
    prefix
}

// see above
enum connection {
    wireguard
    localvxlan
}

// see above for the type definition.
// the key would be a public key, with the private key only available
// locally and stored locally. 
type wireguard {
    // deterministic, based on the public key
    nicname
    // TBD, a peer can be IPv6, IPv6-ll or IPv4
    peer
    // public key
    key
    // Gateways in IPv6 are link-local. To be able to use IPv6 in any way,
    // an interface needs an IPv6 link-local address. As wireguard interfaces 
    // are l3-only, the kernel doesn't assign one, so we need to assign one
    // ourselves. (we need to come up with a deterministic way, so we can be
    // sure we now which/where)
    ll_fe80
}

// definition for later usage
// an l2vxlan wil be connected to a default bridge that gets attached to the
// network resource. That way we can easily add a vxlan to that bridge for 
// local interconnectivity
type l2vxlan {
    // deterministic or stored... 
    nicname
    // Or it's through fdb entries 
    Option<peer>
    // Or it's in a multicast vxlan
    Option<group>
    // a vxlan always has an ID
    id
}

```

### QUESTIONS:

  - Storage of network objects.
  It has been said that we will use the 'blockchain' to store 'things'. A network/network resource could very well be such a 'thing', but at the moment there is no clear view on how to do that and how big these 'things' can be. Also, that 'thing' needs to be encrypted to make sure we don't enlarge a possible attack surface towards a tenant network.
  - notification of change of network objects to interested parties. 
  In case we can use the blockchain to store things, how will other nodes be notified a new network resource is created so that tunnels and routing can be built to establish connectivity to all network resources of a tenant network
  - static routing 
  with all these network resources, static routing can get very hairy very fast. I still think a dynamic routing protocol will make life a lot easier. The advantage is also that routing protocols can detect unavailable links and update routes and/or circumvent the broken link.