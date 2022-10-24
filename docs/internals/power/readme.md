# Power Manager
> Please check issues [here](https://github.com/threefoldtech/home/issues/1303)

> This is to spec implementation details of power manger service as proposed [here](https://github.com/threefoldtech/home/issues/1303#issuecomment-1283716997)

## Introduction
power manager service `powerd` needs to run continuously and provide the following functionality:
- Elections of power management node. This node is the one that will never power off (on it's own) or accept any
power down calls from other nodes
- Identification by neighbor nodes.
- Accept calls to power off by an authenticated node
- Send power down calls to neighbor nodes if (and only if) you are the power manager.
- Send WOL calls to nodes that need to wake up.

This leads us to the following http endpoints definitions

- `/self` return a signed information about self. used mainly for node identifications and elections
- `/power` accept power management calls (mainly off)

## HTTP endpoint:
Those custom headers must be available in all POST requests. And also returned by all endpoints.

- `x-timestamp` this is used instead of the standard date header since most clients set the date header automatically. without giving the caller a chance to set it explicitly. the timestamp is part of the signature
- `x-signature` is the hex encoded signature of the:
  - `timestamp` as set in the x-timestamp header (string)
  - `sha256(body)` as raw bytes.

receivers of the responses of all calls must then verify that the signature matches and the timestamp is in acceptable delay range. An old timestamp means calls might have been tempered

This is needed to verify nodes identities (also callers) in a non SSL environment hence the nodes can't generated trusted ssl certificates (may be look into generating trusted ssl certificates base on the node ed25519 keys)

### `GET /self`
Accepts `NO` Input.

Returns:
```json
{
    "id": "<node-id>",
    "farm": "<farm-id>",
    "address": "<chain-address>",
    "access": true, // if node is public node or not.
}
```

### `POST /power`
Accepts input
```json
{
    "leader": "<node-id>",
    "node": "<node-id>", // the targeted node id, need to match the node id that receive the request
    "target": "down", // can be other targets
}
```

## Elections
Elections are a continues process. Nodes will keep checking their neighbors (say every 10 min). A node can then figure out 2 states:
- Am i a leader, hence never accept power-off requests
- Am i a follower, hence accept power-of requests from any verified neighbor.

### Elections cycle
- On boot, if the node has public config (access node). Nothing need to be done. you are a leader and no need for elections
- if not, follow along the process
- The node fetches the list of all farms in the node. for each node in the list
 - if public key is not yet known, fetch from the grid
 - fetch private ips (zos interface, and mac addresses)
- for each node, call the remote node `/self` verify the node identity has to be done over the zos interface IP. this is to make sure node lives on the same LAN segment.
- Once list of nodes is scanned. the local node will then have a list of nodes that lives on the same lan:
  - If any of the nodes are "public" (has public config) then you lost the elections, you become a follower
  - If any of the nodes has lower `node-id` then you lost the elections too.
  - If you have the lowest `node-id` in the list or no other nodes are reachable. then you become a leader.

## Leader
- A leader never power off.
- A leader listens always to all Power-Up events for all nodes in the farm and try to push a WOL
- A leader always listens to Power-Down events for all nodes in he farm. If an event was received, A request is sent to the designated node to handle the request.

## Follower
- The follower should not handle events but a follower can also forward all the "up" events (by generating WOL packets)
- A follower should accept the power down requests from authenticated neighbors.

## On Power Off
If a node receives a valid power off request from an authenticated node, the node need to
- Set it's current state on the chain to `Down(leader-id)`
- Power off

# Remarks
- A node never powers off itself (without a request from a valid node).
- On boot if a node has both it's target, and current state to `Down` it needs to send an `uptime` to the chain, then power off immediately
- On boot if a node has it's target state to `up` and current state is `down`. the current state need to be set to `Up`
