# K3s on TF Grid

## WHY

We want to be able to provision nodes with k3s installed and configured so that we can have a kubernetes cliusyer deployed easily within the grid

## HOW

A special reservation process will setup nodes with the correct binaries of [k3s](https://github.com/rancher/k3s) and with the proper initialisation so that

- We have the necessary kube config to access the cluster from a client. Indeed once k3s setup a k3s.yaml file will be generated on the master node and we need it to configure the cluster on a client machine (usually at ~/.kube/config)

- With kubectl installed on the client machine we can ask for the nodes of our cluster

```
$ kubectl get nodes
NAME        STATUS     ROLES    AGE   VERSION
zv2k8s-03   NotReady   <none>   20d   v1.16.3-k3s.2
zv2k8s-01   Ready      master   20d   v1.16.3-k3s.2
zv2k8s-04   Ready      <none>   20d   v1.16.3-k3s.2
zv2k8s-02   Ready      <none>   20d   v1.16.3-k3s.2
```

at this point we have a master nodes and worker nodes communicationg with each other.

## WHAT

**What do we have so far**

We have been able to deploy several containers including a drupal application connected to a mysql database [ressources files available /ressources/drupal-mysql](/ressources/drupal-mysql) and a wordpress with its mysql server [ressources files available /ressources/wordpress](/ressources/wordpress).

We also have deployed through HELM charts prometheus and grafana monitoring of the cluster

```
$ helm install --namespace mon --name prometheus  stable/prometheus-operator
$ helm list
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART
        APP VERSION
prometheus      mon             1               2019-12-17 18:13:08.296637106 +0100 CET deployed        prometheus-operator-8.3.3  0.34.0
```

We can deploy

- [x] k8s: Create and Deploy basic resources (Pods, multi container pods, deployment, replicaset, secrets, configmap )
- [x] k8s: Create Services with clusterIP and NodePort
- [x] k8s: Create PV and PVC with local path
- [x] k8s: Create and Deploy prometheus monitoring with helm
- [ ] k8s: Create and Deploy Storage solution (PV, PVC)
  - [x] local path
  - [x] longhorn :boom: iscsiadm/open-iscsi must be installed on the host
  - [x] Rook
  - [x] Rook NFS :boom: need nfs-common
  - [x] Rook CEPH :boom: need rdb module and higher kernel
  - [x] Rook cockroach
  - [x] Rook minio
- [ ] k8s: Create and Deploy Ingress Controllers
- [ ] k8s: Create and Deploy an HA cluster
- [ ] k8s: Create and Deploy cert manager with helm
- [ ] k8s: Create and Deploy applications (CI/CD, multi tier)
- [ ] k8s: Test monitoring
- [ ] k8s: Test logging
- [ ] k8s: Test security
- [ ] k8s: Final report

**What is needed to have a production ready kubernetes cluster**

- networking
  - network policies
  - Test different CNI provider
  - ingress
  - automatic https certifcation with traeffik
- secrets
  - encryption at rest: Kubernetes API encrypts the secrets (optionally, using an external KMS system) before storing them in etcd.
- storage
  - decentralized storage
  - NFS
- high availability setup
  - HA PROXY and metalLB

## High Availability

![k3s ha architecture](ressources/ha/k3s-ha-architecture.svg)

**Fixed Registration Address for Agent Nodes**

In the high-availability server configuration, each node must also register with the Kubernetes API by using a fixed registration address, as shown in the diagram below.

After registration, the agent nodes establish a connection directly to one of the server nodes.

Agent nodes are registered with a websocket connection initiated by the k3s agent process, and the connection is maintained by a client-side load balancer running as part of the agent process.

Agents will register with the server using the node cluster secret along with a randomly generated password for the node, stored at /etc/rancher/node/password. The server will store the passwords for individual nodes at /var/lib/rancher/k3s/server/cred/node-passwd, and any subsequent attempts must use the same password.

If the /etc/rancher/node directory of an agent is removed, the password file should be recreated for the agent, or the entry removed from the server.

A unique node ID can be appended to the hostname by launching K3s servers or agents using the --with-node-id flag.

![k3s production setup](ressources/ha/k3s-production-setup.svg)

the datastore-endpoint parameter for etcd has the following format:

https://etcd-host-1:2379,https://etcd-host-2:2379,https://etcd-host-3:2379

The above assumes a typical three node etcd cluster. The parameter can accept one more comma separated etcd URLs.

### How to

generate a cluster token and launch the first node with --cluster-init

```
K3S_TOKEN=Ok875Tfs9974MLs9 k3s server --cluster-init
```

### Run with an external DB

The High Availibility setup will lie on the resiliency of the external DB

![k3s ha with external db](ressources/ha/ha_with_external_db.png)

We also have to provide a fixed registration address like shown in the picture above so that the nodes have a single address to contact the masters.

e.g.

```
curl -sfL https://get.k3s.io | \
INSTALL_K3S_EXEC=" \
server \
--write-kubeconfig-mode 644 \
-t SECRET  \
--datastore-endpoint mysql://kadmin:kadmin-pass@tcp(mydb:3306)/k3sdb
INSTALL_K3S_VERSION="v1.0.0" \
sh -
```

### Run with embedded distributed SQLite (dqlite)

![k3s ha with dqlite ](ressources/ha/ha_with_dqlite.png)

The high availibility setup of the kubernetes database relies on the distributed version of sqlite embbeded in the k3s binary
While this feature is currently experimental, we expect it to be the primary architecture for running HA K3s clusters in the future.

We still need a fixed registration address. An idea would be to use the ip of a load blalancer service running on the master nodes

to try

```
$ cd k3s/ressources/ha/docker
$ docker-compose up
```

if server 2and 3 stop it is due to `depends_on`. Indeed `depends_on` does not wait for db and redis to be “ready” before starting web - only until they have been started.
So you just have to start again the container when k3s on server1 has launched

to test the cluster status

```
$ docker-compose exec server1 kubectl get nodes
NAME           STATUS     ROLES    AGE   VERSION
f07c8895b5c0   Ready      <none>   19m   v1.16.3-k3s.2
335b03212d03   Ready      master   18m   v1.16.3-k3s.2
0741c53d1a2f   Ready      master   19m   v1.16.3-k3s.2
a20cc1c09d17   Ready      master   18m   v1.16.3-k3s.2
```

Note that there is some problems when the dqlite leader goes down the remaining nodes fails to elect a new leader.

### LoadBalancer and EXTERNAL IP

#### Klipper

K3s comes with [klipper](https://github.com/rancher/klipper-lb). It assigns the IP of the node to the loadBalancer service. This works by using a host port for each service load balancer and setting up iptables to forward the request to the cluster IP.

This means that klipper can't provision other IP for that we need metalLB

#### Metal LB

The idea is to have a fix IP to connect to that can handle load balancing between the pods. We use metallb to provision IP on demand for LoadBalancer services.
We can apply the first two files in the metallb folder

```
$ cd metallb
$ kubectl create -f 1-metallb.yaml
$ kubectl create -f 2-config_and_registration_service.yaml
```

this will install metallb on our cluster and configure it via a configmap
and lastly it will create a service of type loadbalancer with a fix IP `192.168.168.168`

### Registration address

To achieve a fix registration address we can use a nginx reverse proxy to do a round robin load balancing
on all our servers. See the example in the docker compose

or we can run HAProxy to get a fix ip for all our servers

Note that

## Public IP

### TCP router

we need to initiate a connection from the cluster to an external machine with a public IP. We direct the tunnel to the traefik (or any other ingress controller) service IP

```
$ kg svc -n kube-system
NAME             TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                                     AGE
kube-dns         ClusterIP      10.43.0.10      <none>        53/UDP,53/TCP,9153/TCP                      6d5h
metrics-server   ClusterIP      10.43.36.33     <none>        443/TCP                                     6d5h
traefik          LoadBalancer   10.43.178.254   172.31.1.50   80:32015/TCP,443:32432/TCP,8080:30137/TCP   6d5h
```

We can use TCP Router to do that

```
trc -local 172.31.1.50:80 -remote zaibon.be:8082 -secret coucou
```

but the test have shown that it is too slow

We tried with inlets

```
inlets client  --remote "zaibon.be:8080"  --upstream "http://172.31.1.50:80"  --token "${AUTHTOKEN}"
```

and the performance is good. If you have deployed the drupal-mysql example which contains an ingress ressources you can modify the ingress to the domain name here it is zaibon.be so that now when you hit http://zaibon.be you are redirected to the drupal website

## CNI

Container networking is the mechanism through which containers can optionally connect to other containers, the host, and outside networks like the internet.
The idea behind the CNI initiative is to create a framework for dynamically configuring the appropriate network configuration and resources when containers are provisioned or destroyed.

### Benchmark

**Ressources consumption**

![CNI Ressources consumption](ressources/cni/ressource_consumption.png)

**Performance**

![CNI Performance](ressources/cni/CNIPerf.png)

**Network Policies and Encryption**

![Network Policies and Encryption](ressources/cni/networkpolicies_encryption.png)

### Flannel by default

It is one of the most mature examples of networking fabric for container orchestration systems, intended to allow for better inter-container and inter-host networking.

From an administrative perspective, it offers a simple networking model that sets up an environment that’s suitable for most use cases when you only need the basics.

By default, K3s will run with flannel as the CNI.
The default backend for flannel is VXLAN. We can enable encryption by passing the IPSec (Internet Protocol Security) or WireGuard options .

### Calico

Project Calico, or just Calico, is another popular networking option in the Kubernetes ecosystem. While Flannel is positioned as the simple choice, Calico is best known for its performance, flexibility, and power. Calico takes a more holistic view of networking, concerning itself not only with providing network connectivity between hosts and pods, but also with network security and administration. The Calico CNI plugin wraps Calico functionality within the CNI framework.

calico kernel requirement

- nf_conntrack_netlink subsystem
- ip_tables (for IPv4)
- ip6_tables (for IPv6)
- ip_set
- xt_set
- ipt_set
- ipt_rpfilter
- ipt_REJECT
- ipip (if using Calico networking)

Unlike Flannel, Calico does not use an overlay network. Instead, Calico configures a layer 3 network that uses the BGP routing protocol to route packets between hosts. This means that packets do not need to be wrapped in an extra layer of encapsulation when moving between hosts. The BGP routing mechanism can direct packets natively without an extra step of wrapping traffic in an additional layer of traffic.

In addition to networking connectivity, Calico is well-known for its advanced network features. Network policy is one of its most sought after capabilities. In addition, Calico can also integrate with Istio, a service mesh, to interpret and enforce policy for workloads within the cluster both at the service mesh layer and the network infrastructure layer. This means that you can configure powerful rules describing how pods should be able to send and accept traffic, improving security and control over your networking environment.

Run K3s with --flannel-backend=none
./k3s server --flannel-backend=none &

## Storage

### Local Path k3s storage class

When deploying an application that needs to retain data, you’ll need to create persistent storage. Persistent storage allows you to store application data external from the pod running your application. This storage practice allows you to maintain application data, even if the application’s pod fails.

A persistent volume (PV) is a piece of storage in the Kubernetes cluster, while a persistent volume claim (PVC) is a request for storage.

![Persistant storage in kubernetes](ressources/storage/simple-localpath/persistentstorage.png)

PV has three access modes

- RWO: Read Write Once. It can only be read/write on one node at any given time
- RWX: Read Write Many. It can only be read/write on multiple node at the same time
- ROX: Read Only Many

K3s comes with Rancher’s Local Path Provisioner and this enables the ability to create persistent volume claims out of the box using local storage on the respective node.
StorageClass "local-path": Only support ReadWriteOnce access mode

let's create a hostPath backed persistent volume claim and a pod to utilize it:

```
cd ressources/storage/simple-localpath/
kubectl create -f pvc.yaml
kubectl create -f pod.yaml
```

### longhorn

K3s supports Longhorn. Longhorn is an open-source distributed block storage system for Kubernetes.
Apply the longhorn.yaml to install Longhorn:

```
cd ressources/storage/longhorn/
kubectl create -f longhorn.yaml
kubectl create -f sc.yaml
```

Problem

```
[longhorn-manager-lp7v2] time="2020-01-08T09:52:05Z" level=error msg="Failed environment check, please make sure you have iscsiadm/open-iscsi installed on the host"
[longhorn-manager-lp7v2] time="2020-01-08T09:52:05Z" level=fatal msg="Error starting manager: Environment check failed: Failed to execute: nsenter [--mount=/host/proc/1/ns/mnt --net=/host/proc/1/ns/net iscsiadm --version], output nsenter: failed to execute iscsiadm: No such file or directory\n, error exit status 1"
```

### Rook

Rook is an open source cloud-native storage orchestrator, providing the platform, framework, and support for a diverse set of storage solutions to natively integrate with cloud-native environments.

Rook turns storage software into self-managing, self-scaling, and self-healing storage services. It does this by automating deployment, bootstrapping, configuration, provisioning, scaling, upgrading, migration, disaster recovery, monitoring, and resource management. Rook uses the facilities provided by the underlying cloud-native container management, scheduling and orchestration platform to perform its duties.

installing [rook](https://rook.io/docs/rook/v1.2/)

```
git clone --single-branch --branch release-1.2 https://github.com/rook/rook.git
cd cluster/examples/kubernetes/ceph
kubectl create -f common.yaml
kubectl create -f operator.yaml
kubectl create -f cluster-test.yaml
```

### Installing [rook NFS](https://rook.io/docs/rook/v1.2/nfs.html)

NFS allows remote hosts to mount filesystems over a network and interact with those filesystems as though they are mounted locally. This enables system administrators to consolidate resources onto centralized servers on the network.

#### First deploy the Rook NFS operator using the following commands:

```
$ cd resources/storage/rook-NFS
$ kubectl create -f 1-operator.yaml
```

We will create a NFS server instance that exports storage that is backed by the default StorageClass. In k3s environments storageClass "local-path": Only support ReadWriteOnce access mode so for the PVC taht must be created before creating NFS CRD instance.

```
$ kubectl create -f 2-nfs.yaml
$  kubectl get pvc -n rook-nfs
NAME                STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
nfs-default-claim   Bound    pvc-9804c4f1-80e1-45ec-bf05-3dbdd012564e   1Gi        RWO            local-path     3m49s
$ kubectl get po -n rook-nfs
NAME         READY   STATUS    RESTARTS   AGE
rook-nfs-0   1/1     Running   0          16s
```

#### Accessing the Export through dynamic NFS provisioning

Once the NFS Operator and an instance of NFSServer is deployed. A storageclass has to be created to dynamically provisioning volumes.
The StorageClass need to have the following 3 parameters passed.

- exportName: It tells the provisioner which export to use for provisioning the volumes.
- nfsServerName: It is the name of the NFSServer instance.
- nfsServerNamespace: It namespace where the NFSServer instance is running.

```
$ kubectl create -f 3-sc.yaml
```

Once the above storageclass has been created create a PV claim referencing the storageclass as shown in the example given below.

```
$ kubectl create -f 4-pvc.yaml
```

#### Consuming the Export

Now we can consume the PV that we just created by creating an example web server app that uses the above `PersistentVolumeClaim` to claim the exported volume. There are 2 pods that comprise this example:

- A web server pod that will read and display the contents of the NFS share
- A writer pod that will write random data to the NFS share so the website will continually update
  Start both the busybox pod (writer) and the web server from the ressources/storage/rook-NFS folder:

```
kubectl create -f busybox-rc.yaml
kubectl create -f web-rc.yaml
```

CANT CREATE POD SUCK IN CONTAINER CREATING

```
  Warning  FailedMount       15m                   kubelet, zv2k8s-04  Unable to attach or mount volumes: unmounted volumes=[rook-nfs-vol], unattached volumes=[default-token-f6sx2 rook-nfs-vol]: timed out waiting for the condition
  Warning  FailedMount       2m31s (x10 over 16m)  kubelet, zv2k8s-04  MountVolume.SetUp failed for volume "pvc-903d405d-0c9c-4af7-bc2d-356fc03905fb" : mount failed: exit status 255
Mounting command: mount
Mounting arguments: -t nfs 10.43.102.171:/nfs-default-claim /var/lib/kubelet/pods/91d477ac-7e3f-4957-b427-0a3f2a68847b/volumes/kubernetes.io~nfs/pvc-903d405d-0c9c-4af7-bc2d-356fc03905fb
```

probably need nfs-common

### Installing [rook CEPH](https://rook.io/docs/rook/v1.2/ceph.html)

RBD

Rook Ceph requires a Linux kernel built with the RBD module. Many distributions of Linux have this module but some don’t, e.g. the GKE Container-Optimised OS (COS) does not have RBD. You can test your Kubernetes nodes by running modprobe rbd. If it says ‘not found’, you may have to rebuild your kernel or choose a different Linux distribution.

CephFS

If you will be creating volumes from a Ceph shared file system (CephFS), the recommended minimum kernel version is 4.17. If you have a kernel version less than 4.17, the requested PVC sizes will not be enforced. Storage quotas will only be enforced on newer kernels.

No RDB module and kernel < 4.17

```
usr/lib/modules/4.14.82-Zero-OS/kernel/drivers # find /usr/lib/modules | grep rdb
/usr/lib/modules/4.14.82-Zero-OS/kernel/drivers/media/rc/keymaps/rc-avermedia-cardbus.ko
```

### Installing [rook CockroachDB](https://rook.io/docs/rook/v1.2/cockroachdb.html)

**Deploy CockroachDB Operator**
First deploy the Rook CockroachDB operator using the following commands:

```
cd resources/storage/rook-cockroachdb
kubectl create -f operator.yaml
```

You can check if the operator is up and running with:

```
 kubectl -n rook-cockroachdb-system get pod
```

**Create and Initialize CockroachDB Cluster**

```
kubectl create -f cluster.yaml
kubectl -n rook-cockroachdb get clusters.cockroachdb.rook.io
```

To check if all the desired replicas are running, you should see the same number of entries from the following command as the replica count that was specified in cluster.yaml:

```
kubectl -n rook-cockroachdb get pod -l app=rook-cockroachdb
```

**Accessing the Database**
To use the cockroach sql client to connect to the database cluster, run the following command in its entirety:

```
kubectl -n rook-cockroachdb-system exec -it $(kubectl -n rook-cockroachdb-system get pod -l app=rook-cockroachdb-operator -o jsonpath='{.items[0].metadata.name}') -- /cockroach/cockroach sql --insecure --host=cockroachdb-public.rook-cockroachdb
```

This will land you in a prompt where you can begin to run SQL commands directly on the database cluster.

Example:

```
root@cockroachdb-public.rook-cockroachdb:26257/> show databases;
+----------+
| Database |
+----------+
| system   |
| test     |
+----------+
(2 rows)

Time: 2.105065ms
```

**Example App**
If you want to run an example application to exercise your new CockroachDB cluster, there is a load generator application in the same directory as the operator and cluster resource files. The load generator will start writing random key-value pairs to the database cluster, verifying that the cluster is functional and can handle reads and writes.

The rate at which the load generator writes data is configurable, so feel free to tweak the values in loadgen-kv.yaml. Setting --max-rate=0 will enable the load generator to go as fast as it can, putting a large amount of load onto your database cluster.

To run the load generator example app, simply run:

```
kubectl create -f loadgen-kv.yaml
```

You can check on the progress and statistics of the load generator by running:

```
 kubectl -n rook-cockroachdb logs -l app=loadgen
```

To connect to the database and view the data that the load generator has written, run the following command:

```
kubectl -n rook-cockroachdb-system exec -it $(kubectl -n rook-cockroachdb-system get pod -l app=rook-cockroachdb-operator -o jsonpath='{.items[0].metadata.name}') -- /cockroach/cockroach sql --insecure --host=cockroachdb-public.rook-cockroachdb -d test -e 'select * from kv'
```

### Installing [rook minio](https://rook.io/docs/rook/v1.2/minio-object-store.html)

**Deploy Minio Operator**

First deploy the Rook CockroachDB operator using the following commands:

```
cd resources/storage/rook-cockroachdb
kubectl create -f operator.yaml
```

You can check if the operator is up and running with:

```
 kubectl -n rook-cockroachdb-system get pod
```

**Create and Initialize a Distributed Minio Object Store**
Now that the operator is running, we can create an instance of a distributed Minio object store by creating an instance of the objectstore.minio.rook.io resource. Some of that resource’s values are configurable, so feel free to browse object-store.yaml and tweak the settings to your liking.

It is strongly recommended to update the values of accessKey and secretKey in object-store.yaml to a secure key pair, as described in the Minio client quickstart guide.

When you are ready to create a Minio object store, simply run:

```
kubectl create -f object-store.yaml
kubectl -n rook-minio get objectstores.minio.rook.io
kubectl -n rook-minio get pod -l app=minio,objectstore=my-store
```

**Accessing the Object Store**

Minio comes with an embedded web based object browser. In the example, the object store we have created can be exposed external to the cluster at the Kubernetes cluster IP via a “NodePort”. We can see which port has been assigned to the service via:

```
kubectl -n rook-minio get service minio-my-store -o jsonpath='{.spec.ports[0].nodePort}'
```

then navigate to the ip of a node and the port printed with the line above

![minio webui](ressources/storage/rook-minio/miniowebui.png)
