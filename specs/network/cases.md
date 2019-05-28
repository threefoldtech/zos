
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

// the exit container(ns) holda as well a prefix as netresource as well as
// an IPv6 address that is going to hold the routes for all prefixes of the
// network. That IPv6 address will thus be the gateway for all prefixes that
// are part of that network. That also means that an upstream router needs to
// know the list of prefixes that need to be routed to that IPv6 address.
type exit {
    netresource
    ipv4_conf
    ipv6_conf
}

type ipv4_conf {
    ipaddress // cidr
    gateway
    metric
    interface
    nat
}

type ipv6_conf {
    ipaddress // cidr
    gateway
    metric
    interface
}

type connected {
    connection
    prefix
}

enum connection {
    wireguard
    localvxlan
    localgre
}

type wireguard {
    nicname
    peer
    key
    ll_fe80
}

type l2vxlan {
    nicname
    peer
    id
    Option<group>
    ll_fe80
}

```