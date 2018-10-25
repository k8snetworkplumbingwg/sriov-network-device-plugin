# SRIOV Network device plugin for Kubernetes
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

 4. Create SRIOV network device plugin Pod
 ```
$ kubectl create -f pod-sriovdp.yaml
```

 5. Get a bash terminal to the SRIOV network device plugin Pod
 ```
$ kubectl exec -it sriov-device-plugin bash
```

 6. Execute the SRIOV network device plugin binary from within the Pod
````
$ ./usr/bin/sriovdp --logtostderr -v 10

sriov-device-plugin.go:380] SRIOV Network Device Plugin started...
sriov-device-plugin.go:190] Discovering SRIOV network device[s]
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp0s31f6/device/sriov_numvfs
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp14s0/device/sriov_numvfs
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp5s0f0/device/sriov_numvfs
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp5s0f1/device/sriov_numvfs
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp5s0f2/device/sriov_numvfs
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp5s0f3/device/sriov_numvfs
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp6s2/device/sriov_numvfs
sriov-device-plugin.go:92] Checking for file /sys/class/net/enp6s2f1/device/sriov_numvfs
sriov-device-plugin.go:121] Sriov Capable Path: /sys/class/net/enp5s0f0/device/sriov_totalvfs
sriov-device-plugin.go:133] Total number of VFs for device enp5s0f0 is 32
sriov-device-plugin.go:135] SRIOV capable device discovered: enp5s0f0
sriov-device-plugin.go:148] Number of Configured VFs for device enp5s0f0 is 2
sriov-device-plugin.go:171] PCI Address for device enp5s0f0, VF 0 is 0000:06:02.0
sriov-device-plugin.go:171] PCI Address for device enp5s0f0, VF 1 is 0000:06:02.1
sriov-device-plugin.go:121] Sriov Capable Path: /sys/class/net/enp5s0f1/device/sriov_totalvfs
sriov-device-plugin.go:133] Total number of VFs for device enp5s0f1 is 32
sriov-device-plugin.go:135] SRIOV capable device discovered: enp5s0f1
sriov-device-plugin.go:148] Number of Configured VFs for device enp5s0f1 is 0
sriov-device-plugin.go:121] Sriov Capable Path: /sys/class/net/enp5s0f2/device/sriov_totalvfs
sriov-device-plugin.go:133] Total number of VFs for device enp5s0f2 is 32
sriov-device-plugin.go:135] SRIOV capable device discovered: enp5s0f2
sriov-device-plugin.go:148] Number of Configured VFs for device enp5s0f2 is 0
sriov-device-plugin.go:121] Sriov Capable Path: /sys/class/net/enp5s0f3/device/sriov_totalvfs
sriov-device-plugin.go:133] Total number of VFs for device enp5s0f3 is 32
sriov-device-plugin.go:135] SRIOV capable device discovered: enp5s0f3
sriov-device-plugin.go:148] Number of Configured VFs for device enp5s0f3 is 0
sriov-device-plugin.go:195] Starting SRIOV Network Device Plugin server at: /var/lib/kubelet/device-plugins/sriovNet.sock
sriov-device-plugin.go:220] SRIOV Network Device Plugin server started serving
sriov-device-plugin.go:402] SRIOV Network Device Plugin registered with the Kubelet
sriov-device-plugin.go:291] ListAndWatch: send devices &ListAndWatchResponse{Devices:[&Device{ID:0000:06:02.0,Health:Healthy,} &Device{ID:0000:06:02.1,Health:Healthy,}],}
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
