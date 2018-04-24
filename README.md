# SRIOV Network device plugin for Kubernetes
## Table of Contents

- [SRIOV Network device plugin](#sriov-network-device-plugin)
- [Prerequisites](#prerequisites)
	-  [Supported SRIOV NICs](#supported-sriov-nics)
- [Quick Start](#quick-start)
	- [Network Object CRDs](#network-object-crds)
	- [Meta-Plugin CNI](#meta-plugin-cni) 
	 - [SRIOV CNI](#sriov-cni)
	 - [Build and run SRIOV Device plugin and CNI-Shim](#build-and-run-sriov-device-plugin-and-cni-shim)
	 - [Testing SRIOV workloads](#testing-sriov-workloads)  
		 - [Deploy test Pod](#deploy-test-pod)
		 - [Verify Pod network interfaces](#verify-pod-network-interfaces)
		 - [Verify Pod routing table](#verify-pod-routing-table)		 
- [Issues and Contributing](#issues-and-contributing)

## SRIOV Network Device Plugin
The goal of the SRIOV Network device plugin is to manage the lifecycle of SRIOV VFs on a Kubernetes node. The device plugin discovers, advertises and allocates SRIOV VFs to requesting pods and the SRIOV CNI using information passed from the CNI Shim plumbs the VF to the pods network namespace. 
- Device Plugin/Device Manager

  - Discovery of SRIOV NIC devices in a node

  - Advertisement of number of SRIOV VFs configured on a node

  - Allocation of SRIOV VF to a pod

  - Storing of VF Information to Pod Information

- CNI Shim

  - Establish gRPC communication with device plugin

  - Retrieve VF information for a pod from device plugin using gRPC

  - Passing retrieved information from device plugin to SRIOV CNI

- SRIOV CNI

  -  On Cmd Add, using information passed from CNI Shim plumb allocated SRIOV VF to the pods network namespace

  - On Cmd Del, release VF from the pods network namespace

This implementation follows the directions of [this proposal document](https://docs.google.com/document/d/1Ewe9Of84GkP0b2Q2PC0y9RVZNkN2WeVEagX9m99Nrzc/).
## Prerequisites
There are list of items should be required before installing the SRIOV Network device plugin
 1.  SRIOV NICs - (Tested with Intel NIC, should support all the NICs)
 2.  Intel SRIOV CNI - v0.3 Alpha
 3. Kubernetes version - 1.10 (with patch applied included in [`patches/`]() )
 4. Meta plugin - Multus v2.0
 5. Enable `--feature-gates=DevicePlugins=True` in the Kubelet

Make sure to implement the steps described in [Quick Start](#quick-start) for Kubernetes cluster to support multi network.  SRIOV network device plugin is a collective plugin model to work with device plugin, Meta-plugin and SRIOV CNI plugin.
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
This section explains how to set up SRIOV Network device plugin in Kubernetes. Required YAML files can be found in [deployments/](deployments/) directory.
### Network Object CRDs
Kubernetes out of the box only allows to have one network interface per pod. In order to add multiple interfaces in a Pod we need to configure Kubernetes with a CNI meta plugin that enables invoking multiple CNI plugins to add additional interfaces.  [Multus](https://github.com/intel/multus-cni) is only meta plugin that supports this mechanism. Multus uses Kubernetes Custom Resource Definition or CRDs to define network objects. For more information see Multus [documentation](https://github.com/intel/multus-cni/blob/master/README.md). 
### Meta Plugin CNI
1. Compile Meta Plugin CNI (Multus v2.0):
````
$ git clone --branch v2.0 https://github.com/intel/multus-cni.git --single-branch
$ cd multus-cni
$ ./build
$ cp bin/multus /opt/cni/bin
````
2. Configure Kubernetes network CRD with [Multus](https://github.com/intel/multus-cni#usage-with-kubernetes-crdtpr-based-network-objects)

### SRIOV CNI
 Compile SRIOV-CNI:

    $ git clone https://github.com/Intel-Corp/sriov-cni.git
    $ git fetch
    $ git checkout dev/sriov-network-device-plugin-alpha
    $ cd sriov-cni
    $ ./build
    $ cp bin/sriov /opt/cni/bin

#### Build and run SRIOV Device plugin and CNI-Shim

 1. Clone the sriov-network-device-plugin repository
 ```
$ git clone https://github.com/intel/sriov-network-device-plugin.git
 ```  
 2. Run the build script, this will build the SRIOV Network Device Plugin binaries as well as generated the protobuf API for gRPC communication 
 ``` 
$ ./build.sh
```      
 2. Copy the CNI Shim binary from the bin folder to the CNI binary folder
```
$ cp bin/cnishim /opt/cni/bin
```   
 3. Copy the CNI Shim Configuration file from the Deployments folder to the CNI Configuration diectory
```
$ cp deployments/cni-conf.json /etc/cni/net.d/
```

>Note: ensure the CNI Shim configuration file is the first file in lexicographical order in the folder 
 4. Create the CNI-Shim Network CRD
```
$ kubectl create -f deployments/cnishim-crd.yaml
```
 
 5. Run build docker script to create SRIOV Network Device Plugin Docker image
 ```
$ cd deployments/
$ ./build_docker.sh
``` 
 6. Create SRIOV Network Device Plugin Pod
 ```
$ kubectl create -f pod-sriovdp.yaml
```
 >Note: This is for demo purposes, the SRIOV Device Plugin binary must be executed from within the pod

 7. Get a bash terminal to the SRIOV Network Device Plugin Pod
 ```
$ kubectl exec -it sriov-device-plugin bash
```

 8. Execute the SRIOV Network Device Plugin binary from within the Pod
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
