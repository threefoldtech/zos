# zos networking

## Boot and initial setup

At boot, be it from an usb stick or PXE, ZOS starts up the kernel, with a few
necessary parameters like farmerid and/or possible network parameters, but
basically once the kernel has started, zinit among other things, starts the
network initializer.

In short, that process loops over the available network interfaces and tries to
obtain an IP address that also provides for a default gateway. That means: it
tries to get Internet connectivity. Without it, ZOS stops there, as not being able
to register itself, nor start other processes, there wouldn't be any use for it
to be started anyway.

Once it has obtained Internet connectivity, ZOS can then proceed to make itself
known to the Grid, and acknowledge it's existence. It will then regularly poll
the Grid for tasks.

Once initialized, with the network daemon running (a process that will handle
all things related to networking), ZOS will set up some basic services so that
workloads can themselves use that network.

## networkd functionality

The network daemon is in itself responsible for a few tasks, and working
together with the provision daemon it mainly sets up the local infrastructure to
get the user network resources, together with the wireguard configurations for
the user's mesh network.

The Wireguard mesh is an overlay network. That means that traffic of that network
is encrypted and encapsulated in a new traffic frame that the gets transferred
over the underlay network, here in essence the network that has been set up
during boot of the node.

For users or workloads that run on top of the mesh, the mesh network looks and
behaves like any other directly connected workload, and as such that workload
can reach other workloads or services in that mesh with the added advantage
that that traffic is encrypted, protecting services and communications over
that mesh from too curious eyes.

That also means that workloads between nodes in a local network of a farmer is
even protected from the farmer himself, in essence protecting the user from the
farmer in case that farmer could become too curious.

As the nodes do not have any way to be accessed, be it over the underlaying
network  or even the local console of the node, a user can be sure that his
workload cannot be snooped upon.

## Techie talk

- **boot and initial setup**
For ZOS to work at all (the network is the computer), it needs an internet
connection. That is: it needs to be able the BCDB over the internet.  
So ZOS starts with that: with the `internet` process, that tries go get the node to receive an IP address. That process will have set-up a bridge (`zos`), connected to an interface that is on an Internet-capable network. That bridge will have an IP address that has Internet access.
Also, that bridge is there for future public interfaces into workloads.  
Once ZOS can reach the Internet, the rest of the system can be  started, where ultimately, the `networkd` daemon is started.
- **networkd initial setup**
`networkd` starts with recensing the available Network interfaces, and registers them to the BCDB (grid database), so that farmers can specify non-standard configs like for multi-nic machines. Once that is done, `networkd` registers itself to the zbus, so it can receive tasks to execute from the provsioning daemon (`provisiond`).  
These tasks are mostly setting up network resources for users, where a network resource is a subnet in the user's wireguard mesh.

- multi-nic setups
- registering and configurations
- farmer considerations

## wireguard explanations

- wireguard as pointopoint links and what that means
- wireguard underlay usage
- wireguard port management
- wireguard and hidden nodes

## caveats

- hidden nodes
- local underlay network reachability
- IPv6 and IPv4 considerations

## future

- CNI
- automated provisioning
- fully routable IPv6 to your mesh
-




