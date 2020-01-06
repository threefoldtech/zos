# tfuser CLI

tfuser is a small CLI tool that we created to help developer deploy workload on a 0-OS node during development.

It has 3 main commands:

- **id**: use to generate user identity
- **generate**: used to provisioning reservation schema
- **provision**: used to send schema created with the generate command to a node

## Examples

### Generate a network provisioning schema

Here is a full example how you would use tfuser to create a network schema.

1. create a network from scratch
`tfuser generate network create --name {NETWORK_NAME} --cidr ${CIDR}`

`NETWORK_NAME`: Arbitrary name of your network.
`CIDR`: The prefix which will be used by the network resources in the network.

The schema will be printed on stdout, and should be saved for further actions.

2. add another node to the network:
`tfuser generate --schema ${SCHEMA_NAME} network add-node --node {NODE_ID} --subnet ${SUBNET} --port ${WGPORT}`

`SCHEMA_NAME`: The file name of in which the current schema is stored. The new node
	will be added to the schema, and the new schema version will be saved in the file.
`NODE_ID`: The ID of the node to add in the reservation.
`SUBNET`: The subnet to assign to the network resource on this node. All workloads
	in this network resource will receive an IP from this subnet. The subnet must be
	part of the `CIDR` provided when the network is created.
`WGPORT`: Optional wireguard listening port on the host. If this is not specified, one will be
	generated automatically. It is the responsibility of the caller to ensure this port
	is not already in use.

The result of these two command should looks like:
```json
{
  "id": "",
  "user_id": "",
  "type": "network",
  "data": {
    "name": "suitnet",
    "net_id": "",
    "ip_range": "172.20.0.0/16",
    "net_resources": [
      {
        "node_id": "qzuTJJVd5boi6Uyoco1WWnSgzTb7q8uN79AjBT9x9N3",
        "subnet": "172.20.1.0/24",
        "wg_private_key": "53955637e985a7763ac45c3d08cc0e40d64bfc1806db79bc9b0c092b3479567dffb42eda4ae460cc22131e4d254895b89af0e8acdd97a2fd87cfd4e12be8c23d617bd2c93f86aa1eaab35aa8cf57327de6a4dfb8066490cbc06e51de",
        "wg_public_key": "ZdSn/3gDfD1urtDCOPdOFwF0XsTAidJTSpwPETegZGw=",
        "wg_listen_port": 12345,
        "peers": [
          {
            "subnet": "172.20.2.0/24",
            "wg_public_key": "TxC9PcNEBpNtwAvWePRA5/jQz4musN0aayQ3Jk6w3AU=",
            "allowed_ips": [
              "172.20.2.0/24",
              "100.64.20.2/32"
            ],
            "endpoint": ""
          },
          {
            "subnet": "172.20.3.0/24",
            "wg_public_key": "tA0tYq2Z+tIeG1oW+1QJ/cJN6FpDX17ogwf3SefaGjs=",
            "allowed_ips": [
              "172.20.3.0/24",
              "100.64.20.3/32"
            ],
            "endpoint": "[2a02:1802:5e:0:225:90ff:fef4:2917]:12347"
          },
          {
            "subnet": "172.20.4.0/24",
            "wg_public_key": "0VcC6PCOW4JDZERtOUljSKXiPfd0di5qRQ/SHFn9XCs=",
            "allowed_ips": [
              "172.20.4.0/24",
              "100.64.20.4/32"
            ],
            "endpoint": "[2a02:1802:5e:0:225:90ff:fef4:40af]:12348"
          }
        ]
      },
      {
        "node_id": "Gr8NxBLHe7yjSsnSTgTqGr7BHbyAUVPJqs8fnudEE4Sf",
        "subnet": "172.20.2.0/24",
        "wg_private_key": "73eaebb93d9efe7bc7e3efaabcb757b6f115abfcbf24dedfbd26952d228eeb7a65e9f5138c1dc6490a8155dfd126750e712856485f9731b12aecf819855db0e0efac34029142f1179ef62bb7bbdac49439810b04a3bb8fe4b696e3a4",
        "wg_public_key": "TxC9PcNEBpNtwAvWePRA5/jQz4musN0aayQ3Jk6w3AU=",
        "wg_listen_port": 12346,
        "peers": [
          {
            "subnet": "172.20.1.0/24",
            "wg_public_key": "ZdSn/3gDfD1urtDCOPdOFwF0XsTAidJTSpwPETegZGw=",
            "allowed_ips": [
              "172.20.1.0/24",
              "100.64.20.1/32",
              "172.20.3.0/24",
              "100.64.20.3/32",
              "172.20.4.0/24",
              "100.64.20.4/32"
            ],
            "endpoint": "185.69.166.246:12345"
          }
        ]
      },
      {
        "node_id": "48YmkyvfoXnXqEojpJieMfFPnCv2enEuqEJcmhvkcdAk",
        "subnet": "172.20.3.0/24",
        "wg_private_key": "c5d25d7990110fdd8613e129b89af294427feba69d2bb6e8d460eaf68dff030952f070b220308ddb9e58035545335cd14fa89bb4094cdc2aa32a8a7a82b3d398ec739a2f23148fa9478984329daa6083d5e222660352537444730b7c",
        "wg_public_key": "tA0tYq2Z+tIeG1oW+1QJ/cJN6FpDX17ogwf3SefaGjs=",
        "wg_listen_port": 12347,
        "peers": [
          {
            "subnet": "172.20.1.0/24",
            "wg_public_key": "ZdSn/3gDfD1urtDCOPdOFwF0XsTAidJTSpwPETegZGw=",
            "allowed_ips": [
              "172.20.1.0/24",
              "100.64.20.1/32",
              "172.20.2.0/24",
              "100.64.20.2/32"
            ],
            "endpoint": "[2a02:1802:5e:0:1000::f6]:12345"
          },
          {
            "subnet": "172.20.4.0/24",
            "wg_public_key": "0VcC6PCOW4JDZERtOUljSKXiPfd0di5qRQ/SHFn9XCs=",
            "allowed_ips": [
              "172.20.4.0/24",
              "100.64.20.4/32"
            ],
            "endpoint": "[2a02:1802:5e:0:225:90ff:fef4:40af]:12348"
          }
        ]
      },
      {
        "node_id": "BpTAry1Na2s1J8RAHNyDsbvaBSM3FjR4gMXEKga3UPbs",
        "subnet": "172.20.4.0/24",
        "wg_private_key": "4ca059feb0831ed84dfa5a436fcbfa3a23e9b57766b1a1a2d4167a0f7407265010077008b689b6926cc13cd2383db30102435df0c75b2dca2989a42e307972c4ec4b9860f38c0684aea69e5b6d6b40a051320de86ac5b3536d8c619f",
        "wg_public_key": "0VcC6PCOW4JDZERtOUljSKXiPfd0di5qRQ/SHFn9XCs=",
        "wg_listen_port": 12348,
        "peers": [
          {
            "subnet": "172.20.1.0/24",
            "wg_public_key": "ZdSn/3gDfD1urtDCOPdOFwF0XsTAidJTSpwPETegZGw=",
            "allowed_ips": [
              "172.20.1.0/24",
              "100.64.20.1/32",
              "172.20.2.0/24",
              "100.64.20.2/32"
            ],
            "endpoint": "[2a02:1802:5e:0:1000::f6]:12345"
          },
          {
            "subnet": "172.20.3.0/24",
            "wg_public_key": "tA0tYq2Z+tIeG1oW+1QJ/cJN6FpDX17ogwf3SefaGjs=",
            "allowed_ips": [
              "172.20.3.0/24",
              "100.64.20.3/32"
            ],
            "endpoint": "[2a02:1802:5e:0:225:90ff:fef4:2917]:12347"
          }
        ]
      }
    ]
  },
  "created": "0001-01-01T00:00:00Z",
  "duration": 0,
  "to_delete": false
}
```

#### Network provisioning graph

Once you have a reservation schema of a network, it is possible to create a
dot file of the schema, which can then be turned into an image of the network
graph. This shows how different network resources will reach each other.

`tfuser generate --schema ${SCHEMA} network graph`

Will print the dot file on stdout. You can then use your preferred layout engine
to generate the image.

Example image generated from the schema above:

![Example network graph](example_graph.png)

### Generate a container provisioning schema

```shell
tfuser generate --schema ubuntu.json container \
--flist https://hub.grid.tf/tf-official-apps/ubuntu-bionic-build.flist \
--corex \
--entrypoint /bin/bash \
--network ZKhTrpy7tubzG5fxqdsWzwoXbEhtwJ1yxDBz8xDUf9J \
--env KEY:VALUE
```

### Provision a workload on a node

To provision a workload on a node, you need to have a user identity. It is used to sign the schema before sending to the node.  
Generate an identity using `tfuser id`. This command will generate a `user.seed` file that you need to use with the provision command.


```shell
tfuser provision --schema container.json --duration 2 --seed user.seed --node {nodeID}
```
