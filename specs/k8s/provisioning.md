# K8s provisioning on 0-OS

Reservation schema:

```go
type K8sCluster struct {
    // address of the master node, if empty, then we are the master
    // and need to start k3s as server otherwise stars as agent and connect
    // to this address
    MasterAddr string
    // authentication token
    Token string
}
```

What happens when a k8s reservation arrives on a node ?

- check that there is no "regular" workloads current provisioned.
- provisiond switch to a state where is refused to provision any additional workloads, the node is now fully reserved
- download k3s binaries and prepare directory on cache disk for data directory of k3s
- starts k3s binary in server or agent mode depending on the reservation schema content