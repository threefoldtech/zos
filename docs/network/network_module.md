# Network module

## ZBus

Network module is available on zbus over the following channel

| module | object | version |
|--------|--------|---------|
| network|[network](#interface)| 0.0.1|

## Home Directory

network keeps some data in the following locations
| directory | path|
|----|---|
| root| `/var/cache/modules/network`|

## Interface

```go
//Networker is the interface for the network module
type Networker interface {
  ApplyNetResource(Network) (string, error)
  DeleteNetResource(Network) error
  Namespace(NetID) (string, error)
}
```

## Zero-OS networking

### Some First Explanations

To start, ZOS needs networking, as without it, there is no point to start.
Therefore, the `internet` process gets started, and that:

- configures the Node's initial network configuration, so that the Node can register itself.  For now we assume that the Node is connected to a network (ethernet segment) that provides IP addresses over DHCP, be it IPv4 or IPv6, or that there is a Routing Avertisement (RA) daemon for IPv6 running on that network.  
Only once it has received an IP Address, most other internal services will be able to start. ([John Gage](https://www.networkcomputing.com/cloud-infrastructure/network-computer-again) from Sun said that `The Network is the Computer`, here that is absolutely true)
If it doesn't succeed it's bootstrap, nothing else will, and 0-OS will stop there.
- Notifies [zinit](https://github.com/threefoldtech/zinit/blob/master/docs/readme.md) (the services orchestrator in 0-OS) that it can register the dhcp client as a permanent process on the intitially discovered NIC (Network Interface Card) and that zinit can start other processes, one of which takes care of registering the node to the grid. (more elaborate explanation about that in [identity service](../identity/readme.md)).
- `zinit` then can start the `networkd` daemon that will monitor the `zbus` for network tasks

- `networkd` will then listen in on the zbus for new or updated Network Resources (NR) that get sent by the provision daemon and applies them.

### Jargon

So. Let's have some abbreviations settled first:

- #### Node : simple  

  TL;DR: Computer.

  A Node is a computer with CPU, Memory, Disks (or SSD's, NVMe) connected to _A_ network that has Internet access. (i.e. it can reach www.google.com, just like you on your phone, at home)  
  That Node will, once it has received an IP address (IPv4 or IPv6), register itself when it's new, or confirm it's identity and it's online-ness (for lack of a better word).

- #### TNo : Tenant Network object. [The gory details here](https://github.com/threefoldtech/zos/blob/master/pkg/network.go)
  
  TL;DR: The Network Description.

  We named it so, because it is a datastructure that describes the __whole__ network a user can request (or setup).  
  That network is a virtualized overlay network.  
  Basically that means that transfer of data in that network *always* is encrypted, protected from prying eyes, and __resources in that network can only communicate with each other__ **unless** there is a special rule that allows access. Be it by allowing accesss through firewall rules, *and/or* through a proxy (a service that forwards requests on behalf of, and ships replies back to the client).

- #### A Tno has an ExitPoint (for IPv6)
  
  TL;DR: Any network needs to get out *somewhere*. [Some more explanation](exitpoints.md)

  A Node that happens to live in an Internet Network (to differentiate from a Tenant network), more explictly, a network that is directly routable and accessible (unlike a home network), can be specified as an Exit Node.  
  That Node can then host Exitpoints for Tenant Networks.  
  Let's explain that.  
  Entities in a Tenant Network, where a TN being an overlay network, can only communicate with peers that are part of that network. At a certain point there is a gateway needed for this network to communicate with the 'external' world (BBI): that is an ExitPoint. ExitPoints can only live in Nodes designated for that purpose, namely Exit Nodes. Exit Nodes can only live in networks that are bidirectionally reachable for THE Internet (BBI).  
  An ExitPoint is *always* a part of a Network Resource (see below).

- #### Network Resource: (NR)
  
  TL;DR: the Node-local part of a TNo.

  The main building block of a TNo; i.e. each service of a user in a Node lives in an NR.  
  Each Node hosts User services, whatever type of service that is. Every service in that specific node will always be solely part of the Tenant's Network. (read that twice).  
  So: A Network Resource is the thing that interconnects all other network resources of the TN (Tenant Network), and provides routing/firewalling for these interconnects, including the default route to the BBI (Big Bad Internet).  
  