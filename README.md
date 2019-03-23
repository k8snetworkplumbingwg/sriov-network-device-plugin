# SRIOV Network device plugin for Kubernetes

[![Travis CI](https://travis-ci.org/intel/sriov-network-device-plugin.svg?branch=release-v1)](https://travis-ci.org/intel/sriov-network-device-plugin)

## Table of Contents

- [SRIOV Network device plugin](#sriov-network-device-plugin)
	-  [Supported SRIOV NICs](#supported-sriov-nics)
- [Quick Start](#quick-start)
	- [Network Object CRDs](#network-object-crds)
	- [Build and configure Multus](#build-and-configure-multus) 
	- [Build SRIOV CNI](#build-sriov-cni)
	- [Build and run SRIOV network device plugin](#build-and-run-sriov-network-device-plugin)
	- [Testing SRIOV workloads](#testing-sriov-workloads)
		 - [Deploy test Pod](#deploy-test-pod)
		 - [Verify Pod network interfaces](#verify-pod-network-interfaces)
		 - [Verify Pod routing table](#verify-pod-routing-table)		 
- [Issues and Contributing](#issues-and-contributing)

## SRIOV Network Device Plugin
The SRIOV network device plugin is Kubernetes device plugin for discovering and advertising SRIOV network virtual functions (VFs) in a Kubernetes host. To deploy workloads with SRIOV VF this plugin needs to work together with the following two CNI plugins:

- Multus CNI

  - Retrieves allocated network device information of a Pod

  - Passes allocated SRIOV VF information to SRIOV CNI

- SRIOV CNI

  - During Pod creation, plumbs allocated SRIOV VF to a Pods network namespace using VF information given by Multus

  - On Pod deletion, reset and release the VF from the Pod

This implementation follows the design discuessed in [this proposal document](https://docs.google.com/document/d/1Ewe9Of84GkP0b2Q2PC0y9RVZNkN2WeVEagX9m99Nrzc/).


Please follow the Multus [Quick Start](#quick-start) for multi network interface support in Kubernetes.

### Supported SRIOV NICs
The following  NICs were tested with this implementation. However, other SRIOV capable NICs should work as well.
-  Intel® Ethernet Controller X710 Series 4x10G
		- PF driver : v2.4.6
		- VF driver: v3.5.6
> please refer to Intel download center for installing latest [Intel Ethernet Controller-X710-Series](https://downloadcenter.intel.com/product/82947/Intel-Ethernet-Controller-X710-Series) drivers
 - Intel® 82599ES 10 Gigabit Ethernet Controller
	- PF driver : v4.4.0-k
	- VF driver: v3.2.2-k
> please refer to Intel download center for installing latest [Intel-® 82599ES 10 Gigabit Ethernet](https://ark.intel.com/products/41282/Intel-82599ES-10-Gigabit-Ethernet-Controller) drivers

## Quick Start
This section explains an exmaple deployment of SRIOV Network device plugin in Kubernetes. Required YAML files can be found in [deployments/](deployments/) directory.

### Network Object CRDs
Multus uses Custom Resource Definitions(CRDs) for defining additional network attachements. These network attachment CRDs follow the standards defined by K8s Network Plumbing Working Group(NPWG). Please refer to [Multus documentation](https://github.com/intel/multus-cni/blob/master/README.md) for more information.

### Build and configure Multus
1. Compile Multus executable:
```
$ git clone https://github.com/intel/multus-cni.git
$ cd multus-cni
$ ./build
$ cp bin/multus /opt/cni/bin
```
 2. Copy the multus Configuration file from the Deployments folder to the CNI Configuration diectory
```
$ cp deployments/cni-conf.json /etc/cni/net.d/
```

3. Configure Kubernetes network CRD with [Multus](https://github.com/intel/multus-cni/tree/dev/network-plumbing-working-group-crd-change#creating-network-resources-in-kubernetes)
```
$ kubectl create -f deployments/crdnetwork.yaml
```

### Build SRIOV CNI
 1. Compile SRIOV-CNI (dev/k8s-deviceid-model branch):
```
$ git clone https://github.com/intel/sriov-cni.git
$ cd sriov-cni
$ git fetch
$ git checkout dev/k8s-deviceid-model
$ ./build
$ cp bin/sriov /opt/cni/bin
```
 2. Create the SRIOV Network CRD
```
$ kubectl create -f deployments/sriov-crd.yaml
```
### Build and run SRIOV network device plugin

 1. Clone the sriov-network-device-plugin
 ```
$ git clone https://github.com/intel/sriov-network-device-plugin.git
$ cd sriov-network-device-plugin
 ```  
 2. Build executable binary using `make` 
 ``` 
$ make
```      
> On successful build the `sriovdp` executable can be found in `./build` directory. It is recommended to run the plugin in a container or K8s Pod. The follow on steps cover how to build and run the Docker image of the plugin.

 3. Build docker image
 ```
$ make image
``` 

 4. Create SRIOV network device plugin Pod and the SRIOV network device plugin binary will be exectued from within the Pod
 ```
$ kubectl create -f pod-sriovdp.yaml
```

 5. Check for the logs of the SRIOV network device plugin binary from within the pod
````
$ kubectl logs sriov-device-plugin

I0323 05:49:12.547174  324440 sriov-device-plugin.go:469] Starting SRIOV Network Device Plugin...
I0323 05:49:12.550968  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/cni0
I0323 05:49:12.551082  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/cni0/device/sriov_numvfs
I0323 05:49:12.551159  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/docker0
I0323 05:49:12.551224  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/docker0/device/sriov_numvfs
I0323 05:49:12.551328  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/eno1
I0323 05:49:12.551445  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/eno1/device/sriov_numvfs
I0323 05:49:12.551846  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/eno2
I0323 05:49:12.551922  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/eno2/device/sriov_numvfs
I0323 05:49:12.551999  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s0f0
I0323 05:49:12.552074  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s0f0/device/sriov_numvfs
I0323 05:49:12.552152  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s0f1
I0323 05:49:12.552225  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s0f1/device/sriov_numvfs
I0323 05:49:12.552306  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s2
I0323 05:49:12.552382  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s2/device/sriov_numvfs
I0323 05:49:12.552463  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s2f1
I0323 05:49:12.552538  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s2f1/device/sriov_numvfs
I0323 05:49:12.552623  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s2f2
I0323 05:49:12.552698  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s2f2/device/sriov_numvfs
I0323 05:49:12.552777  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s2f3
I0323 05:49:12.552852  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s2f3/device/sriov_numvfs
I0323 05:49:12.552930  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s2f4
I0323 05:49:12.553010  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s2f4/device/sriov_numvfs
I0323 05:49:12.553090  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp135s2f5
I0323 05:49:12.553165  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp135s2f5/device/sriov_numvfs
I0323 05:49:12.553243  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp175s0f0
I0323 05:49:12.553321  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp175s0f0/device/sriov_numvfs
I0323 05:49:12.553403  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp175s0f1
I0323 05:49:12.553478  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp175s0f1/device/sriov_numvfs
I0323 05:49:12.553556  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp177s0f0
I0323 05:49:12.553635  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp177s0f0/device/sriov_numvfs
I0323 05:49:12.553714  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/enp177s0f1
I0323 05:49:12.553791  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/enp177s0f1/device/sriov_numvfs
I0323 05:49:12.553870  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/lo
I0323 05:49:12.553942  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/lo/device/sriov_numvfs
I0323 05:49:12.554015  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/veth389adcb
I0323 05:49:12.554087  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/veth389adcb/device/sriov_numvfs
I0323 05:49:12.554159  324440 sriov-device-plugin.go:106] Checking inside dir /sys/class/net/vethb391bc4b
I0323 05:49:12.554231  324440 sriov-device-plugin.go:117] Checking for file /sys/class/net/vethb391bc4b/device/sriov_numvfs
I0323 05:49:12.554307  324440 sriov-device-plugin.go:209] Discovering all capable and configured devices
I0323 05:49:12.554340  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/eno1/device/sriov_totalvfs
I0323 05:49:12.554489  324440 sriov-device-plugin.go:230] Total number of VFs for device eno1 is 32
I0323 05:49:12.554523  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: eno1
I0323 05:49:12.554641  324440 sriov-device-plugin.go:245] Number of Configured VFs for device eno1 is 0
I0323 05:49:12.554675  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/eno2/device/sriov_totalvfs
I0323 05:49:12.554780  324440 sriov-device-plugin.go:230] Total number of VFs for device eno2 is 32
I0323 05:49:12.554812  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: eno2
I0323 05:49:12.554916  324440 sriov-device-plugin.go:245] Number of Configured VFs for device eno2 is 0
I0323 05:49:12.554944  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/enp135s0f0/device/sriov_totalvfs
I0323 05:49:12.555044  324440 sriov-device-plugin.go:230] Total number of VFs for device enp135s0f0 is 64
I0323 05:49:12.555075  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: enp135s0f0
I0323 05:49:12.555204  324440 sriov-device-plugin.go:245] Number of Configured VFs for device enp135s0f0 is 6
I0323 05:49:12.557289  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/enp135s0f1/device/sriov_totalvfs
I0323 05:49:12.557439  324440 sriov-device-plugin.go:230] Total number of VFs for device enp135s0f1 is 64
I0323 05:49:12.557485  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: enp135s0f1
I0323 05:49:12.557645  324440 sriov-device-plugin.go:245] Number of Configured VFs for device enp135s0f1 is 0
I0323 05:49:12.557690  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/enp175s0f0/device/sriov_totalvfs
I0323 05:49:12.557819  324440 sriov-device-plugin.go:230] Total number of VFs for device enp175s0f0 is 8
I0323 05:49:12.557861  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: enp175s0f0
I0323 05:49:12.558021  324440 sriov-device-plugin.go:245] Number of Configured VFs for device enp175s0f0 is 0
I0323 05:49:12.558075  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/enp175s0f1/device/sriov_totalvfs
I0323 05:49:12.558230  324440 sriov-device-plugin.go:230] Total number of VFs for device enp175s0f1 is 8
I0323 05:49:12.558276  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: enp175s0f1
I0323 05:49:12.558421  324440 sriov-device-plugin.go:245] Number of Configured VFs for device enp175s0f1 is 0
I0323 05:49:12.558470  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/enp177s0f0/device/sriov_totalvfs
I0323 05:49:12.558635  324440 sriov-device-plugin.go:230] Total number of VFs for device enp177s0f0 is 6
I0323 05:49:12.558680  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: enp177s0f0
I0323 05:49:12.558821  324440 sriov-device-plugin.go:245] Number of Configured VFs for device enp177s0f0 is 0
I0323 05:49:12.558861  324440 sriov-device-plugin.go:218] Sriov Capable Path: /sys/class/net/enp177s0f1/device/sriov_totalvfs
I0323 05:49:12.558978  324440 sriov-device-plugin.go:230] Total number of VFs for device enp177s0f1 is 6
I0323 05:49:12.559007  324440 sriov-device-plugin.go:232] SRIOV capable device discovered: enp177s0f1
I0323 05:49:12.559092  324440 sriov-device-plugin.go:245] Number of Configured VFs for device enp177s0f1 is 0
I0323 05:49:12.559119  324440 sriov-device-plugin.go:262] Discovered SR-IOV PF devices: [enp135s0f0]
I0323 05:49:12.559166  324440 sriov-device-plugin.go:308] Starting SRIOV Network Device Plugin server at: /var/lib/kubelet/plugins_registry/sriovNet.sock
I0323 05:49:12.560940  324440 sriov-device-plugin.go:333] SRIOV Network Device Plugin server started serving
I0323 05:49:12.563297  324440 sriov-device-plugin.go:370] Plugin: sriovNet.sock gets registered successfully at Kubelet
I0323 05:49:12.563499  324440 sriov-device-plugin.go:385] ListAndWatch: send initial devices &ListAndWatchResponse{Devices:[&Device{ID:0000:87:02.3,Health:Healthy,} &Device{ID:0000:87:02.4,Health:Healthy,} &Device{ID:0000:87:02.5,Health:Healthy,} &Device{ID:0000:87:02.0,Health:Healthy,} &Device{ID:0000:87:02.1,Health:Healthy,} &Device{ID:0000:87:02.2,Health:Healthy,}],}
````

### Testing SRIOV workloads
Leave the sriov device plugin running and open a new terminal session for following steps.
#### Deploy test Pod
````
$ kubectl create -f pod-tc1.yaml
pod "testpod1" created

$ kubectl get pods
NAME                  READY     STATUS    RESTARTS   AGE
sriov-device-plugin   1/1       Running   0          7h
testpod1        	  1/1       Running   0          3s
````
#### Verify Pod network interfaces
````
$ kubectl exec -it testpod1 -- ip addr show

1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
3: eth0@if17511: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UP
    link/ether 0a:58:c0:a8:4a:b1 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 192.168.74.177/24 scope global eth0
       valid_lft forever preferred_lft forever
17508: net0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc mq state DOWN qlen 1000
    link/ether ce:d8:06:08:e6:3f brd ff:ff:ff:ff:ff:ff
    inet 10.56.217.179/24 scope global net0
       valid_lft forever preferred_lft forever
````
#### Verify Pod routing table
````
$ kubectl exec -it testpod1 -- route -n

Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         192.168.74.1    0.0.0.0         UG    0      0        0 eth0
10.56.217.0     0.0.0.0         255.255.255.0   U     0      0        0 net0
192.168.0.0     192.168.74.1    255.255.0.0     UG    0      0        0 eth0
192.168.74.0    0.0.0.0         255.255.255.0   U     0      0        0 eth0
````

## Issues and Contributing
We welcome your feedback and contributions to this project. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines. 

Copyright 2018 © Intel Corporation.
