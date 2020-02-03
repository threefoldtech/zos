# Kubernetes cluster provisioning, deploiement and examples on Threefold Grid

In this guide we will walk you through the provisioning of a full-blown kubernetes cluster
on the TF grid.

We will then see how to connect to it and interact using kubectl on our local machine.

Finally we will go through some examples use cases to grasp the features offered by the cluster.

# Provisioning and Deploiement of the cluster

Our CLI tool tfuser makes it easy to interact with the TFGrid and provision a kubernetes cluster.

## Tfuser Install

Tfuser is included in our zos repository. Let's clone the repository and build the binary

```
$ git clone https://github.com/threefoldtech/zos
$ cd tools
$ make tfuser
cd tfuser && go build -ldflags '-w -s -X github.com/threefoldtech/zos/pkg/version.Branch=fcvm -X github.com/threefoldtech/zos/pkg/version.Revision=c00740b8fbb75746cd02fb5614ccf57342b7a9f4 -X github.com/threefoldtech/zos/pkg/version.Dirty='
$ cd tfuser
$ ./tfuser --help
```

## Identity on the TF Grid

To interact with the grid we need an indentity which is simply some cryptographic keys that we will use to sign transaction on the network.
Let's generate them with tfuser

```
$ tfuser id
new identity generated: 7HZd49ugHzJxQUnHKyQRbmcwWWvKKtoJUndfiiXGTkow
seed saved at user.seed
```

NB: a file named user.seed has been created. This file is sensitive as it stores your private key. Make sure not to loose that file.

## Provisioning

To provision our VMs we first need to setup a network between different Nodes on the TFGrid
We will use devnet Nodes to select some nodes go to :[devnet Cockpit](https://cockpit.devnet.grid.tf/)
We picked three nodes identified by these NodeId

- nodeid1: FTothsg9ZuJubAEzZByEgQQUmkWM637x93YH1QSJM242
- nodeid2: 3NAkUYqm5iPRmNAnmLfjwdreqDssvsebj4uPUt9BxFPm
- nodeid3: 2anfZwrzskXiUHPLTqH1veJAia8G6rW2eFLy5YFMa1MP

Becarefull NodeId is case sensitive

### Networking

We now need to create a network between the nodes. Hopefully tfuser can help us generate the configuration file.
Let's first create a network named `kubetest` with a large cidr `173.30.3.0/24` and write that configuration inside a file `network.json`

```
$ ./tfuser generate --schema network.json network create --name kubetest --cidr 173.30.3.0/24
```

now let's add the nodes that we have selected inside our network by specifying the path to our `network.json` network configuration file. We also specify a port for wireguard to use and a subnet to use on the nodes.

```
$ ./tfuser generate --schema network.json network add-node --node qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3  \
				      --subnet 173.30.3.0/24 --port 30018
$ ./tfuser generate --schema network.json network add-node --node 3NAkUYqm5iPRmNAnmLfjwdreqDssvsebj4uPUt9BxFPm \
				      --subnet 173.30.3.0/24 --port 30018
$ ./tfuser generate --schema network.json network add-node --node FTothsg9ZuJubAEzZByEgQQUmkWM637x93YH1QSJM242 \
				      --subnet 173.30.3.0/24 --port 30018
```

We will need an external access to the network to be able to ssh into our VMs.

```
$ ./tfuser generate --schema network.json network add-access --node qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3 \
				      --subnet 173.30.4.0/24 --ip4 > wg.conf
```
We generated the wireguard, a secure and fast VPN, configuration file that we will use here after in this guide.

Let's take a look at a graphical representation of our network

```
$ ./tfuser generate --schema network.json network graph
```

This will generate network.json.dot file. You can copy paste the content on a website like [graphiz](https://dreampuf.github.io/GraphvizOnline) to see the visual representation of the network we defined.

Now we have a nodes in a network definition ready to be provisioned. Let's use our seed to sign the transaction and ask for a 2 days provision.

```
$ ./tfuser provision --schema network.json --duration 2 --seed user.seed \
				      --node qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3 \
				      --node FTothsg9ZuJubAEzZByEgQQUmkWM637x93YH1QSJM242 \
				      --node 3NAkUYqm5iPRmNAnmLfjwdreqDssvsebj4uPUt9BxFPm
Reservation for 48h0m0s send to node qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3  
Resource: /reservations/2406-1
Reservation for 48h0m0s send to node FTothsg9ZuJubAEzZByEgQQUmkWM637x93YH1QSJM242
Resource: /reservations/2407-1
Reservation for 48h0m0s send to node 3NAkUYqm5iPRmNAnmLfjwdreqDssvsebj4uPUt9BxFPm
Resource: /reservations/2408-1
```

you can check the status of the provision with the live command

```

$ ./tfuser live --seed user.seed --end 3000
ID:2406-1 Type:   network expired at:31-Jan-2020 16:18:31state:     ok  network ID:
ID:2407-1 Type:   network expired at:31-Jan-2020 16:18:31state:     ok  network ID:
ID:2408-1 Type:   network expired at:31-Jan-2020 16:18:31state:     ok  network ID:

```

We will now use the wg.conf file previously generated to connect to our network with wireguard a secure and fast vpn.
be sure to install wireguard and then simply run this command to create a network interface and join the defined network

```
$ wg-quick up ./wg.conf
```

We have setup and provision a network on the grid that we can join through wireguard.
Let's now provision some Kubernetes VM on those nodes

### Provision

Let's provision a kubernetes VM on our first node and assign it an IP allowed in the network `kubetest` defined here above.

```
$ ./tfuser generate --schema kube1.json kubernetes --size 1 --network-id kubetest --ip 173.30.1.2 --secret token --node qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3  --ssh-keys "github:zgorizzo69"

$ ./tfuser -d provision --schema kube1.json --duration 2 --seed user.seed --node qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3  
```
You can check the status again with the live command

```
$ ./tfuser live --seed user.seed --end 3000
D:2581-1 Type:kubernetes expired at:02-Feb-2020 17:35:14state:     ok
```

Let's try to connect to our VM


```

```



\$ ./tfuser generate --schema network.json \
 kubernetes --size 1 \
 --network-id kubenet \
 --ip 172.31.2.50 \
 --node 9aDDL13d8Q7fxz1LjNskTpgXEugBevpQmHo3DJcASTa2

```

```
