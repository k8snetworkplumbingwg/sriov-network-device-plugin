
# SRIOV Network device plugin for Kubernetes

[![Travis CI](https://travis-ci.org/intel/sriov-network-device-plugin.svg?branch=master)](https://travis-ci.org/intel/sriov-network-device-plugin/builds) [![Go Report Card](https://goreportcard.com/badge/github.com/intel/sriov-network-device-plugin)](https://goreportcard.com/report/github.com/intel/sriov-network-device-plugin)

## Table of Contents

- [SRIOV Network device plugin](#sriov-network-device-plugin)
- [Features](#features)
  - [Supported SRIOV NICs](#supported-sriov-nics)
- [Quick Start](#quick-start)
  - [Build SRIOV CNI](#build-sriov-cni)
  - [Build and run SRIOV network device plugin](#build-and-run-sriov-network-device-plugin)
  - [Install one compatible CNI meta plugin](#install-one-compatible-cni-meta-plugin)
      - [Option 1 - Multus](#option-1---multus)
        - [Install Multus](#install-multus)
        - [Network Object CRDs](#network-object-crds)
      - [Option 2 - DANM](#option-2---danm)
        - [Install DANM](#install-danm)
        - [Create SR-IOV type networks](#create-sr-iov-type-networks)
- [Configurations](#configurations)
  - [Config parameters](#config-parameters)
  - [Command line arguments](#command-line-arguments)
  - [Assumptions](#assumptions)
  - [Workflow](#workflow)
- [Example deployments](#example-deployments)
    - [Deploy the Device Plugin](#deploy-the-device-plugin)
    - [Deploy SR-IOV workloads when Multus is used](#deploy-sr-iov-workloads-when-multus-is-used)
      - [Deploy test Pod connecting to pre-created SR-IOV network](#deploy-test-pod-connecting-to-pre-created-sr-iov-network)
      - [Verify Pod network interfaces](#verify-pod-network-interfaces)
      - [Verify Pod routing table](#verify-pod-routing-table)
    - [Deploy SR-IOV workloads when DANM is used](#deploy-sr-iov-workloads-when-danm-is-used)
      - [Verify the existence of the example SR-IOV networks](#verify-the-existence-of-the-example-sr-iov-networks)
      - [Connect your networks to existing SR-IOV Device Pools](#connect-your-networks-to-existing-sr-iov-device-pools)
      - [Deploy demo Pod connecting to pre-created SR-IOV networks](#deploy-demo-pod-connecting-to-pre-created-sr-iov-networks)
      - [Verify status and the network connections of the demo Pod](#verify-status-and-the-network-connections-of-the-demo-pod)
    - [Pod device information](#pod-device-information)
- [Issues and Contributing](#issues-and-contributing)

## SRIOV Network Device Plugin

The SRIOV network device plugin is Kubernetes device plugin for discovering and advertising SRIOV network virtual functions (VFs) in a Kubernetes host.

## Features

- Handles SRIOV capable/not-capable devices (NICs and Accelerators alike)
- Supports devices with both Kernel and userspace(UIO and VFIO) drivers
- Allows resource grouping using "Selector"
- User configurable resourceName
- Detects Kubelet restarts and auto-re-register
- Detects Link status (for Linux network devices) and updates associated VFs health accordingly
- Extensible to support new device types with minimal effort if not already supported

To deploy workloads with SRIOV VF this plugin needs to work together with the following two CNI components:

- Any CNI meta plugin supporting Device Plugin based network provisioning (Multus CNI, or DANM)

  - Retrieves allocated network device information of a Pod

- SRIOV CNI

  - During Pod creation, plumbs allocated SRIOV VF to a Pods network namespace using VF information given by the meta plugin

  - On Pod deletion, reset and release the VF from the Pod


Please follow the [Quick Start](#quick-start) for multi network interface support in Kubernetes.

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

- Mellanox ConnectX®-4 Lx EN Adapter
- Mellanox ConnectX®-5 Adapter
> Network card drivers are available as a part of the various linux distributions and upstream.
To download the latest Mellanox NIC drivers, click [here](http://www.mellanox.com/page/software_overview_eth).

## Quick Start

### Build SRIOV CNI

1. Compile SRIOV-CNI (supported from release 2.0+):
```
$ git clone https://github.com/intel/sriov-cni.git
$ cd sriov-cni
$ make
$ cp build/sriov /opt/cni/bin
```

### Build and run SRIOV network device plugin

 1. Clone the sriov-network-device-plugin
 ```
$ git clone https://github.com/intel/sriov-network-device-plugin.git
$ cd sriov-network-device-plugin
 ```
 2. Build docker image binary using `make`
 ```
$ make
```
> On a successful build, a docker image with tag `nfvpe/sriov-device-plugin:latest` will be created. You will need to build this image on each node. Alternatively, you could use a local docker registry to host this image.

 3. Create a ConfigMap that defines SR-IOV resrouce pool configuration
 ```
$ kubectl create -f deployments/configMap.yaml
```
 4. Deploy SRIOV network device plugin Daemonset
```
$ kubectl create -f deployments/k8s-v1.16/sriovdp-daemonset.yaml
```
> For K8s version v1.15 or older use `deployments/k8s-v1.10-v1.15/sriovdp-daemonset.yaml` instead.


### Install one compatible CNI meta plugin
A compatible CNI meta-plugin installation is required for SR-IOV CNI plugin to be able to get allocated VF's deviceID in order to configure it.  

#### Option 1 - Multus

##### Install Multus
Please refer to Multus [Quickstart Installation Guide](https://github.com/intel/multus-cni#quickstart-installation-guide) to install Multus.

##### Network Object CRDs

Multus uses Custom Resource Definitions(CRDs) for defining additional network attachements. These network attachment CRDs follow the standards defined by K8s Network Plumbing Working Group(NPWG). Please refer to [Multus documentation](https://github.com/intel/multus-cni/blob/master/README.md) for more information.
1. Create the SRIOV Network CRD
```
$ kubectl create -f deployments/sriov-crd.yaml
```

#### Option 2 - DANM
This section explains an example deployment of SRIOV Network device plugin in Kubernetes if you choose DANM as your meta plugin.

##### Install DANM
Refer to [DANM documentation](https://github.com/nokia/danm#getting-started) for detailed instructions.

##### Create SR-IOV type networks
DANM supports the Device Plugin based SR-IOV provisioning with the dynamic level.
This means that all DANM API features seamlessly work together with the SR-IOV setup described above, whether you use the [lightweight](https://github.com/nokia/danm#lightweight-network-management-experience), or the [production grade](https://github.com/nokia/danm#production-grade-network-management-experience) network management APIs.
For example manifest objects refer to [SR-IOV demo](https://github.com/nokia/danm/tree/master/example/device_plugin_demo)

> See following sections on how to configure and run SRIOV device plugin.

## Configurations

### Config parameters

This plugin creates device plugin endpoints based on the configurations given in  the config map associated with the SRIOV device plugin. In json format as this files appears as shown below:

```json
{
    "resourceList": [{
            "resourceName": "intel_sriov_netdevice",
            "selectors": {
                "vendors": ["8086"],
                "devices": ["154c", "10ed"],
                "drivers": ["i40evf", "ixgbevf"]
            }
        },
        {
            "resourceName": "intel_sriov_dpdk",
            "selectors": {
                "vendors": ["8086"],
                "devices": ["154c", "10ed"],
                "drivers": ["vfio-pci"],
                "pfNames": ["enp0s0f0","enp2s2f1"]
            }
        },
        {
            "resourceName": "mlnx_sriov_rdma",
            "isRdma": true,
            "selectors": {
                "vendors": ["15b3"],
                "devices": ["1018"],
                "drivers": ["mlx5_ib"]
            }
        },
        {
            "resourceName": "infiniband_rdma_netdevs",
            "isRdma": true,
            "selectors": {
                "linkTypes": ["infiniband"]
            }
        }
    ]
}
```

`"resourceList"` should contain a list of config objects. Each config object may consist of following fields:



|     Field      | Required |        Description        |                      Type - Accepted values                       |                                      Example/Accepted values                                       |
|----------------|----------|---------------------------|-------------------------------------------------------------------|----------------------------------------------------------------------------------------------------|
| "resourceName" | Yes      | Endpoint resource name    | string - must be unique and should not contain special characters | "sriov_net_A"                                                                                      |
| "selectors"    | No       | A map of device selectors | Each selector is a map of string list.                            | "vendors": ["8086"],"devices": ["154c", "10ed"],"drivers": ["vfio-pci"],"pfNames": ["enp2s2f0"],"linkTypes": ["ether"] |
| "isRdma"       | No       | Mount RDMA resources      | `bool` - boolean value true or false                              | "isRdma": true                                                                                     |



[//]: # (The table above generated using: https://ozh.github.io/ascii-tables/)

### Command line arguments

This plugin accepts the following optional run-time command line arguments:

```bash
./sriovdp --help

Usage of ./sriovdp:
  -alsologtostderr
        log to standard error as well as files
  -config-file string
        JSON device pool config file location (default "/etc/pcidp/config.json")
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -resource-prefix string
        resource name prefix used for K8s extended resource (default "intel.com")
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```

### Assumptions

This plugin does not bind or unbind any driver to any device whether it's PFs or VFs. It also doesn't create Virtual functions either. Usually, the virtual functions are created at boot time when kernel module for the device is loaded. Required device drivers could be loaded on system boot-up time by white-listing/black-listing the right modules. But plugin needs to be aware of the driver type of the resources(i.e. devices) that it is registering as K8s extended resource so that it's able to create appropriate Device Specs for the requested resource.

For example, if the driver type is uio(i.e. igb_uio.ko) then there are specific device files to add in Device
Spec. For vfio-pci, device files are different. And if it is Linux kernel network driver then there is no device file to be added.

The idea here is, user creates a resource config for each resource pool as shown in [Config parameters](#config-parameters) by specifying the resource name, a list resource "selectors".

The device plugin will initially discover all PCI network resources in the host and populate an initial "device list". Each "resource pool" then applies its selectors on this list and add devices that satisfies the selectors constraints. Each selector narrows down the list of devices for the resource pool. Currently, the selectors are applied in following order:

1. "vendors" - The vendor hex code of device
2. "devices" - The device hex code of device
3. "drivers" - The driver name the device is registered with
4. "pfNames" - The Physical function name
5. "linkTypes" - The link type of the net device associated with the PCI device.

The "pfName" selector can be used to specify a range of VFs for a pool in the next format:
````
"<PFName>#<FirstVF>-<LastVF>"
````

Where:

    `<PFName>`  - is the PF interface name
    `<FirstVF>` - is the first VF index (0-based) that included into the range
    `<LastVF>`  - is the last VF index (0-based) that included into the range

Example:

The selector for interface named `netpf0` and VF range from 2 upto 7 (included 2 and 7) will look like:
````
"pfName": ["netpf0#2-7"]
````
If only PF network interface specified in the selector, then assuming that all VFs of this interface are going to the pool.


### Workflow

- Load device's (Physical function if it is SRIOV capable) kernel module and bind the driver to the PF
- Create required Virtual functions
- Bind all VF with right drivers
- Create a resource config map
- Run SRIOV device plugin (as daemonset)

On successful run, the allocatable resource list for the node should be updated with resource discovered by the plugin as shown below. Note that the resource name appended with the `-resource-prefix` i.e. `"intel.com/sriov_net_A"`.

```json
$ kubectl get node node1 -o json | jq '.status.allocatable'

{
  "cpu": "8",
  "ephemeral-storage": "169986638772",
  "hugepages-1Gi": "0",
  "hugepages-2Mi": "8Gi",
  "intel.com/sriov_net_A": "8",
  "intel.com/sriov_net_B": "8",
  "memory": "7880620Ki",
  "pods": "1k"
}

```

## Example deployments

We assume that you have working K8s cluster configured with one of the supported meta plugins for multi-network support. Please see [Features](#features) and [Quick Start](#quick-start) sections for more information on required CNI plugins.

### Deploy the Device Plugin
The [images](./images) directory contains example Dockerfile, sample specs along with build scripts to deploy the SRIOV device plugin as daemonset. Please see [README.md](./images/README.md) for more information about the Docker images.

````
# Create ConfigMap
$ kubectl create -f deployments/configMap.yaml
configmap/sriovdp-config created

# Create sriov-device-plugin-daemonset
$ kubectl create -f deployments/k8s-v1.16/sriovdp-daemonset.yaml
serviceaccount/sriov-device-plugin created
daemonset.apps/kube-sriov-device-plugin-amd64 created

$ kubectl -n kube-system get pods
NAMESPACE     NAME                                   READY   STATUS    RESTARTS   AGE
kube-system   kube-sriov-device-plugin-amd64-46wpv   1/1     Running   0          4s

````

### Deploy SR-IOV workloads when Multus is used
There are some example Pod specs and related network CRD yaml files can be found in [deployments](./deployments) directory for a sample deployment with Multus.

Leave the SRIOV device plugin running and open a new terminal session for following steps.

#### Deploy test Pod connecting to pre-created SR-IOV network

````
$ kubectl create -f pod-tc1.yaml
pod "testpod1" created

$ kubectl get pods
NAME                  READY     STATUS    RESTARTS   AGE
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

### Deploy SR-IOV workloads when DANM is used
#### Verify the existence of the example SR-IOV networks

````
[cloudadmin@controller-1 ~]$ kubectl get dnet -n example-sriov
NAME         AGE
management   6s
sriov-a      14m
sriov-b      13m
````

#### Connect your networks to existing SR-IOV Device Pools
The Spec.Options.device_pool mandatory parameter denotes the Device Pool used by the network.
Make sure this parameter is set to the name(s) of your existing SR-IOV Device Pool(s)!

````
[cloudadmin@controller-1 ~]$ kubectl describe node 172.31.3.154 | grep -A8 Allocatable
Allocatable:
 cpu:                          6
 ephemeral-storage:            50189Mi
 hugepages-1Gi:                0
 hugepages-2Mi:                0
 memory:                       249150992Ki
 nokia.k8s.io/exclusive_caas:  16
 nokia.k8s.io/shared_caas:     32k
 nokia.k8s.io/sriov_ens2f1:    32

[cloudadmin@controller-1 ~]$ kubectl describe dnet sriov-a -n example-sriov | grep device_pool
    device_pool:       nokia.k8s.io/sriov_ens2f1
[cloudadmin@controller-1 ~]$ kubectl describe dnet sriov-b -n example-sriov | grep device_pool
    device_pool:       nokia.k8s.io/sriov_ens2f1

````

#### Deploy demo Pod connecting to pre-created SR-IOV networks
First, make sure that your Pod asks appropriate number of Devices from the right Device Pools:

````
[cloudadmin@controller-1 ~]$ grep -B1 sriov_ sriov_pod.yaml
      requests:
        nokia.k8s.io/sriov_ens2f1: '2'
      limits:
        nokia.k8s.io/sriov_ens2f1: '2'
````

Then instantiate the Pod:

````
[cloudadmin@controller-1 ~]$ kubectl create -f sriov_pod.yaml
pod/sriov-pod created
````

#### Verify status and the network connections of the demo Pod

````
[cloudadmin@controller-1 ~]$ kubectl get pod sriov-pod -n example-sriov
NAME        READY   STATUS    RESTARTS   AGE
sriov-pod   1/1     Running   0          111s

[cloudadmin@controller-1 ~]$ kubectl exec -n example-sriov -it sriov-pod -- ip addr show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
3: eth0@if49: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 8950 qdisc noqueue
    link/ether 8a:74:fd:e0:ee:fa brd ff:ff:ff:ff:ff:ff
    inet 10.244.3.8/24 brd 10.244.3.255 scope global eth0
       valid_lft forever preferred_lft forever
9: second_path2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq qlen 1000
    link/ether e2:19:e0:1b:91:44 brd ff:ff:ff:ff:ff:ff
26: first_path1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq qlen 1000
    link/ether 7e:0d:fa:eb:83:8c brd ff:ff:ff:ff:ff:ff
````

### Pod device information

The allocated device information are exported in Container's environment variable. The variable name is `PCIDEVICE_` appended with full extended resource name(e.g. intel.com/sriov etc.) which is capitailzed and any special characters(".", "/") are replaced with underscore("_"). In case of multiple devices from same extended resource pool, the device IDs are delimited with commas(",").

For example, if 2 devices are allocated from `intel.com/sriov` extended resource then the allocated device information will be found in following env variable:
`PCIDEVICE_INTEL_COM_SRIOV=0000:03:02.1,0000:03:04.3`

## Issues and Contributing

We welcome your feedback and contributions to this project. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

Copyright 2018 © Intel Corporation.
