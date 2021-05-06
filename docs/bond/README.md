# Bond interfaces with SR-IOV

The below shows a method to create a pod with 2 SR-IOV interfaces bonded into a single bond interface inside a container using [Bond CNI](https://github.com/intel/bond-cni)

This style of configuration can be used to create workloads with high availability - i.e. with two VFs from different PFs connected to the pod. Note that Bond CNI will only work with interfaces utilising the kernel driver. Workloads that leverage userspace drivers on VFs - such as DPDK workloads - are not covered by this guide or by the Bond CNI.

## Prerequisites
This guide assumes the quick set up of the repo has been followed. That is that Multus, SRIOV CNI and SRIOV Network Device Plugin have been installed.

Additionally at least two SRIOV Virtual Functions of the type 'intel_sriov_netdevice' should be available on a target node.

## Installing Bond CNI
In order to get Bond CNI working in your cluster first clone the repo:

`git clone https://github.com/intel/bond-cni`

Next move to the directory and build the binary:

`cd bond-cni && ./build.sh`

As with all CNI directories this binary will need to be placed in the CNI directory on each node in order to be used by the cluster. By default this directory is at `/opt/cni/bin`

For a single node this can simply be copied:
``cp bin/bond /opt/cni/bin``

For each additional host in the cluster run:
``scp bin/bond <USER>@<HOST_IP>:/opt/cni/bin``

## Network Attachment Definitions
Next create the network attachment definitions. The SRIOV CRD is similar to the one in the main repo - but it omits the IP address management as that will be handled by the bond interface. Note that this isn't obligatory but is standard when bonding interfaces.

``kubectl apply -f sriov-net.yaml``

Next create the bonded network attachment definition:

``kubectl apply -f bond-crd.yaml``

### Pod deployment
With the above network definitions in place the following pod should deploy to a node with 2 SRIOV interfaces available and set up the SRIOV and Bond interfaces correctly.
```
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  annotations:
    k8s.v1.cni.cncf.io/networks: '[
{"name": "sriov-net",
"interface": "net1"
},
{"name": "sriov-net",
"interface": "net2"
},
{"name": "bond-net",
"interface": "bond0"
}]'
spec:  # specification of the pod's contents
  restartPolicy: Never
  containers:
  - name: bond-test
    image: alpine:latest
    command:
      - /bin/sh
      - "-c"
      - "sleep 60m"
    imagePullPolicy: IfNotPresent
    resources:
      requests:
        intel.com/intel_sriov_netdevice: '2'
      limits:
        intel.com/intel_sriov_netdevice: '2'
```

Multus will trigger the above CRDs, sriov-net and bond-net1 in the order in which they're listed. This will result in first the creation of a pod with two SRIOV VFs interfaces named net1 and net2 as specified in the networks annotation. It will then create a bond interface named bond0, using the interfaces named net1 and net2 as specified in the bond network attachment definition.

The pod spec contains explicit naming of each of the interfaces in the networks annotation.

In the above example both VFs used in the bond come from the same VF pool. This does not guarantee that the VFs are from different physical functions, or from different root network cards. In a real world scenario failover would not be guaranteed by the above configuration. Instead separate device pools with specific requests from each would need to be implemented to ensure bonding is performed on different network cards.

Once the pod is up and running the command `kubectl exec -it test-pod -- ip a` should result in output like:

```
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
3: eth0@if155: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1450 qdisc noqueue state UP
    link/ether 42:be:17:a8:86:48 brd ff:ff:ff:ff:ff:ff
    inet 10.244.0.107/24 brd 10.244.0.255 scope global eth0
       valid_lft forever preferred_lft forever
4: bond0: <BROADCAST,MULTICAST,UP,LOWER_UP400> mtu 1500 qdisc noqueue state UP qlen 1000
    link/ether e2:c7:95:85:be:89 brd ff:ff:ff:ff:ff:ff
    inet 10.56.217.2/24 scope global bond0
       valid_lft forever preferred_lft forever
149: net2: <BROADCAST,MULTICAST,UP,LOWER_UP800> mtu 1500 qdisc mq master bond0 state UP qlen 1000
    link/ether e2:c7:95:85:be:89 brd ff:ff:ff:ff:ff:ff
151: net1: <BROADCAST,MULTICAST,UP,LOWER_UP800> mtu 1500 qdisc mq master bond0 state UP qlen 1000
    link/ether e2:c7:95:85:be:89 brd ff:ff:ff:ff:ff:ff
```
We can see the bond interface bond0 with an ip address, and two SRIOV network interfaces named net1 and net2. Both of these interfaces identify bond0 as their master meaning the bond is now working correctly.
