
# SR-IOV Network Device Plugin for Kubernetes

[![Travis CI](https://travis-ci.org/k8snetworkplumbingwg/sriov-network-device-plugin.svg?branch=master)](https://travis-ci.org/k8snetworkplumbingwg/sriov-network-device-plugin/builds) [![Go Report Card](https://goreportcard.com/badge/github.com/k8snetworkplumbingwg/sriov-network-device-plugin)](https://goreportcard.com/report/github.com/k8snetworkplumbingwg/sriov-network-device-plugin) [![Weekly minutes](https://img.shields.io/badge/Weekly%20Meeting%20Minutes-Mon%203pm%20GMT-blue.svg?style=plastic)](https://docs.google.com/document/d/1sJQMHbxZdeYJPgAWK1aSt6yzZ4K_8es7woVIrwinVwI)
[![Coverage Status](https://coveralls.io/repos/github/k8snetworkplumbingwg/sriov-network-device-plugin/badge.svg?branch=master)](https://coveralls.io/github/k8snetworkplumbingwg/sriov-network-device-plugin?branch=master)

## Table of Contents

- [SR-IOV Network Device Plugin for Kubernetes](#sr-iov-network-device-plugin-for-kubernetes)
  - [Table of Contents](#table-of-contents)
  - [SR-IOV Network Device Plugin](#sr-iov-network-device-plugin)
  - [Features](#features)
    - [Supported SR-IOV NICs](#supported-sr-iov-nics)
  - [Quick Start](#quick-start)
    - [Creating SR-IOV Virtual Functions](#creating-sr-iov-virtual-functions)
    - [Install SR-IOV CNI](#install-sr-iov-cni)
    - [Get SR-IOV Network Device Plugin container image](#get-sr-iov-network-device-plugin-container-image)
    - [Install SR-IOV Network Device Plugin](#install-sr-iov-network-device-plugin)
    - [Install one compatible CNI meta plugin](#install-one-compatible-cni-meta-plugin)
  - [Configurations](#configurations)
    - [Config parameters](#config-parameters)
    - [Command line arguments](#command-line-arguments)
    - [Assumptions](#assumptions)
    - [Workflow](#workflow)
  - [Example deployments](#example-deployments)
    - [Deploy the Device Plugin](#deploy-the-device-plugin)
    - [Deploy SR-IOV workloads when Multus is used](#deploy-sr-iov-workloads-when-multus-is-used)
    - [Deploy SR-IOV workloads when DANM is used](#deploy-sr-iov-workloads-when-danm-is-used)
    - [Pod device information](#pod-device-information)
  - [Virtual Deployments Support](#virtual-deployments-support)
    - [Configure Device Plugin extended selectors in virtual environments](#configure-device-plugin-extended-selectors-in-virtual-environments)
  - [CNI plugins in virtual environments](#cni-plugins-in-virtual-environments)
    - [Virtual environments with no iommu](#virtual-environments-with-no-iommu)
  - [Multi Architecture Support](#multi-architecture-support)
  - [Container Device Interface](#container-device-interface)
  - [Issues and Contributing](#issues-and-contributing)

## SR-IOV Network Device Plugin

The SR-IOV Network Device Plugin is Kubernetes device plugin for discovering and advertising networking resources in the
form of:
- SR-IOV virtual functions (VFs)
- PCI physical functions (PFs)
- Auxiliary network devices, in particular Subfunctions (SFs)

which are available on a Kubernetes host

## Features

- Handles SR-IOV capable/not-capable devices (NICs and Accelerators alike)
- Handles PCI backed Auxiliary network devices (**at the moment SFs only**)
- Supports devices with both Kernel and userspace (UIO and VFIO) drivers
- Allows resource grouping using "selector(s)"
- User configurable resourceName
- Detects Kubelet restarts and auto-re-register
- Detects Link status (for Linux network devices) and updates associated VFs health accordingly
- Extensible to support new device types with minimal effort if not already supported
- Works within virtual deployments of Kubernetes that do not have virtualized-iommu support (VFIO No-IOMMU support)

To deploy workloads with SR-IOV VF, Auxiliary network devices or PCI PF, this plugin needs to work together with the following two CNI components:

- Any CNI meta plugin supporting Device Plugin based network provisioning (Multus CNI, or DANM)

  - Retrieves allocated network device information of a Pod

- A CNI capable of consuming the network device allocated to the Pod
  - SR-IOV CNI (for SR-IOV VFs)
    - During Pod creation, plumbs allocated SR-IOV VF to a Pods network namespace using VF information given by the meta plugin
    - On Pod deletion, reset and release the VF from the Pod
  - Host device CNI (for PCI PFs)
    - During Pod creation, plumbs the allocated network device to the Pods network namespace using device information given by the meta plugin
    - On Pod deletion, reset and release the allocated network device from the Pod

Please follow the [Quick Start](#quick-start) for multi network interface support in Kubernetes.

### Supported SR-IOV NICs

The following  NICs were tested with this implementation. However, other SR-IOV capable NICs should work as well.

- Intel® Ethernet 800 Series (E810)
- Intel® Ethernet 700 Series (XL710, X710, XXV710)
- Intel® Ethernet 500 Series (82599, X520, X540, X550)
- Mellanox ConnectX-4®
- Mellanox Connectx-4® Lx EN Adapter
- Mellanox ConnectX-5®
- Mellanox ConnectX-5® Ex
- Mellanox ConnectX-6®
- Mellanox ConnectX-6® Dx
- Mellanox BlueField-2®
- Broadcom NetXtreme-E Series Adapters

## Quick Start

### Creating network functions

Before starting the SR-IOV Network Device Plugin you will need to create desired network functions on your system. Docs below will guide you through that process.

- [Creating SR-IOV Virtual Functions](docs/vf-setup.md)
- [Creating Subfunctions](docs/subfunctions/README.md)

### Install SR-IOV CNI

See the [SR-IOV CNI](https://github.com/k8snetworkplumbingwg/sriov-cni) repository for build and installation instructions. Supported from SR-IOV CNI release 2.0+.

### Get SR-IOV Network Device Plugin container image

#### GitHub

```sh
 docker pull ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin:latest
```

#### Build image locally

```sh
 make image
```

> On a successful build, a docker image with tag `ghcr.io/k8snetworkplumbingwg/sriov-network-device-plugin:latest` will be created. You will need to build this image on each node. Alternatively, you could use a local docker registry to host this image.

### Install SR-IOV Network Device Plugin

#### Deploy config map

Create a ConfigMap that defines SR-IOV resource pool configuration

> Make sure to update the 'config.json' entry in the configMap data to reflect your resource configuration for the device plugin. See [Configurations](#configurations) section for supported configuration parameters.

```sh
 kubectl create -f deployments/configMap.yaml
```

#### Deploy daemonset

```sh
 kubectl create -f deployments/sriovdp-daemonset.yaml
```

### Install one compatible CNI meta plugin

A compatible CNI meta-plugin installation is required for SR-IOV CNI plugin to be able to get allocated VF's deviceID in order to configure it.

#### Option 1 - Multus

##### Install Multus

Please refer to Multus [Quickstart Installation Guide](https://github.com/k8snetworkplumbingwg/multus-cni#quickstart-installation-guide) to install Multus.

##### Network Object CRDs

Multus uses Custom Resource Definitions(CRDs) for defining additional network attachements. These network attachment CRDs follow the standards defined by K8s Network Plumbing Working Group (NPWG). Please refer to [Multus documentation](https://github.com/k8snetworkplumbingwg/multus-cni/blob/master/README.md) for more information.

1. Create the SR-IOV Network CRD

```sh
 kubectl create -f deployments/sriov-crd.yaml
```

#### Option 2 - DANM

This section explains an example deployment of SR-IOV Network Device Plugin in Kubernetes if you choose DANM as your meta plugin.

##### Install DANM

Refer to [DANM deployment documentation](https://github.com/nokia/danm/blob/master/deployment-guide.md) for detailed instructions.

##### Create SR-IOV type networks

DANM supports the Device Plugin based SR-IOV provisioning with the dynamic level.
Refer to the [DAMN User Guide documentation](https://github.com/nokia/danm/blob/master/user-guide.md) for detailed instructions.
For example manifest objects refer to [SR-IOV demo](https://github.com/nokia/danm/tree/master/example/device_plugin_demo)

> See following sections on how to configure and run SR-IOV Network Device Plugin.

## Configurations

### Config parameters

This plugin creates device plugin endpoints based on the configurations given in the config map associated with the SR-IOV Network Device Plugin. In json format this file appears as shown below:

```json
{
    "resourceList": [{
            "resourceName": "intel_sriov_netdevice",
            "selectors": [{
                "vendors": ["8086"],
                "devices": ["154c", "10ed", "1889"],
                "drivers": ["i40evf", "ixgbevf", "iavf"]
            }]
        },
        {
            "resourceName": "intel_sriov_dpdk",
            "resourcePrefix": "intel.com",
            "selectors": [{
                "vendors": ["8086"],
                "devices": ["154c", "10ed", "1889"],
                "drivers": ["vfio-pci"],
                "pfNames": ["enp0s0f0","enp2s2f1"],
                "needVhostNet": true
            }]
        },
        {
            "resourceName": "mlnx_sriov_rdma",
            "resourcePrefix": "mellanox.com",
            "selectors": [{
                "vendors": ["15b3"],
                "devices": ["1018"],
                "drivers": ["mlx5_ib"],
                "isRdma": true
            }]
        },
        {
            "resourceName": "infiniband_rdma_netdevs",
            "selectors": [{
                "linkTypes": ["infiniband"],
                "isRdma": true
            }]
        },
        {
            "resourceName": "ct6dx_vdpa_vhost",
            "selectors": [{
                "vendors": ["15b3"],
                "devices": ["101e"],
                "drivers": ["mlx5_core"],
                "vdpaType": "vhost"
            }]
        },
        {
            "resourceName": "intel_fpga",
            "deviceType": "accelerator",
            "selectors": [{
                    "vendors": ["8086"],
                    "devices": ["0d90"]
            }]
        },
        {
            "resourceName": "bf2_sf",
            "resourcePrefix": "nvidia.com",
            "deviceType": "auxNetDevice",
            "selectors": [{
                "vendors": ["15b3"],
                "devices": ["a2d6"],
                "pfNames": ["p0#1-5"],
                "auxTypes": ["sf"]
            }]
        },
        {
          "resourceName": "intel_sriov_netdevice_additional_env",
          "selectors": [{
            "vendors": ["8086"],
            "devices": ["154c", "10ed", "1889"],
            "drivers": ["i40evf", "ixgbevf", "iavf"]
          }],
          "additionalInfo": {
             "*": {
               "token": "3e49019f-412f-4f02-824e-4cd195944205"
             }
          }
        },
        {
            "resourceName": "brcm_sriov_bnxt",
            "resourcePrefix": "broadcom.com",
            "selectors": [{
                "vendors": ["14e4"],
                "devices": ["1750"],
                "drivers": ["bnxt_en"]
            }],
        },
        {
          "resourceName": "old_selectors_syntax_example",
          "selectors": {
            "vendors": ["8086"],
            "devices": ["154c", "10ed", "1889"],
            "drivers": ["i40evf", "ixgbevf", "iavf"]
          }
	}
    ]
}
```

`"resourceList"` should contain a list of config objects. Each config object may consist of following fields:

|       Field       | Required |                                                              Description                                                               |                     Type/Defaults                     |                         Example/Accepted values                        |
|-------------------|----------|----------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------|------------------------------------------------------------------------|
| "resourceName"    | Y        | Endpoint resource name. Should not contain special characters including hyphens and must be unique in the scope of the resource prefix | string                                                | "sriov_net_A"                                                          |
| "resourcePrefix"  | N        | Endpoint resource prefix name override. Should not contain special characters                                                          | string Default : "intel.com"                          | "yourcompany.com"                                                      |
| "deviceType"      | N        | Device Type for a resource pool.                                                                                                       | string value of supported types. Default: "netDevice" | Currently supported values: "accelerator", "netDevice", "auxNetDevice" |
| "excludeTopology" | N        | Exclude advertising of device's NUMA topology                                                                                          | bool Default: "false"                                 | "excludeTopology": true                                                |
| "selectors"       | N        | Either a single device selector map or a list of maps. The list syntax is preferred. The "deviceType" value determines the device selector options.                                                  | json list of objects or json object. Default: null                   | Example: "selectors": [{"vendors": ["8086"],"devices": ["154c"]}]        |
| "additionalInfo" | N | A map of map to add additional information to the pod via environment variables to devices                                             | json object as string Default: null  | Example: "additionalInfo": {"*": {"token": "3e49019f-412f-4f02-824e-4cd195944205"}} |

Note: "resourceName" must be unique only in the scope of a given prefix, including the one specified globally in the CLI params, e.g. "example.com/10G", "acme.com/10G" and "acme.com/40G" are perfectly valid names.

#### Device selectors

The "selectors" field accepts both a single object and a list of selector objects. While both formats are supported, the list syntax is preferred. When using the list syntax, each selector object is evaluated in the order present in the list. For example, a single object would look like:
```json
"selectors": {"vendors": ["8086"],"devices": ["154c"]}
```
and the list syntax would look like:
```json
"selectors": [{"vendors": ["8086"],"devices": ["154c"]}, {"vendors": ["8086"], "needVhostNet": true}]
```

The list syntax example specifies two selector objects in the list. In this example, all devices specified by the first selector object have vendor ID 8086 and device ID 154c. The second selector object specifies all devices with vendor ID 8086. Since the selector objects are processed in the specified order, devices with vendor ID 8086 and device ID 154c would not have the `needVhostNet: true` applied to them. Contrast this with another similar looking example:
```json
"selectors": [{"vendors": ["8086"], "needVhostNet": true}, {"vendors": ["8086"],"devices": ["154c"]}]
```
In this latter example, the order of the selector objects has been switched. Since all devices that are specified by the second object are also specified by the first object, this makes the second selector object ineffective.

A given selector object comprises of "selectors".  The "deviceType" value determines which selectors are supported for that device. Each selector is evaluated in order as listed in selector tables below.

#### Accelerator devices selectors
These selectors are applicable when "deviceType" is "accelerator".

|     Field      | Required |                Description                |         Type/Defaults         |       Example/Accepted values       |
|----------------|----------|-------------------------------------------|-------------------------------|-------------------------------------|
| "vendors"      | N        | Target device's vendor Hex code as string | `string` list Default: `null` | "vendors": ["8086", "15b3"]         |
| "devices"      | N        | Target Devices' device Hex code as string | `string` list Default: `null` | "devices": ["154c", "1889", "1018"] |
| "drivers"      | N        | Target device driver names as string      | `string` list Default: `null` | "drivers": ["vfio-pci"]             |
| "pciAddresses" | N        | Target device's pci address as string     | `string` list Default: `null` | "pciAddresses": ["0000:03:02.0"]    |
| "acpiIndexes"  | N        | Target device's acpi index as string      | `string` list Default: `null` | "acpiIndexes": ["101"]              |


#### Network devices selectors
These selectors are applicable when "deviceType" is "netDevice" (note: this is default)


|     Field      | Required |                               Description                                |                    Type/Defaults                    |                                     Example/Accepted values                                      |
|----------------|----------|--------------------------------------------------------------------------|-----------------------------------------------------|--------------------------------------------------------------------------------------------------|
| "vendors"      | N        | Target device's vendor Hex code as string                                | `string` list Default: `null`                       | "vendors": ["8086", "15b3"]                                                                      |
| "devices"      | N        | Target Devices' device Hex code as string                                | `string` list Default: `null`                       | "devices": ["154c", "1889", "1018"]                                                              |
| "drivers"      | N        | Target device driver names as string                                     | `string` list Default: `null`                       | "drivers": ["vfio-pci"]                                                                          |
| "pciAddresses" | N        | Target device's pci address as string                                    | `string` list Default: `null`                       | "pciAddresses": ["0000:03:02.0"]                                                                 |
| "acpiIndexes"  | N        | Target device's acpi index as string                                     | `string` list Default: `null`                       | "acpiIndexes": ["101"]                                                                           |
| "pfNames"      | N        | functions from PF matches list of PF names                               | `string` list Default: `null`                       | "pfNames": ["enp2s2f0"] (See follow-up sections for some advance usage of "pfNames")             |
| "rootDevices"  | N        | functions from PF matches list of PF PCI addresses                       | `string` list Default: `null`                       | "rootDevices": ["0000:86:00.0"] (See follow-up sections for some advance usage of "rootDevices") |
| "linkTypes"    | N        | The link type of the net device associated with the PCI device           | `string` list Default: `null`                       | "linkTypes": ["ether"]                                                                           |
| "ddpProfiles"  | N        | A map of device selectors                                                | `string` list Default: `null`                       | "ddpProfiles": ["GTPv1-C/U IPv4/IPv6 payload"]                                                   |
| "isRdma"       | N        | Mount RDMA resources. Incompatible with vdpaType                         | `bool` values `true` or `false` Default: `false`    | "isRdma": `true`                                                                                 |
| "needVhostNet" | N        | Share /dev/vhost-net and /dev/net/tun                                    | `bool` values `true` or `false` Default: `false`    | "needVhostNet": `true`                                                                           |
| "vdpaType"     | N        | The type of vDPA device (virtio, vhost). Incompatible with isRdma = true | `string` values `vhost` or `virtio` Default: `null` | "vdpaType": "vhost"                                                                              |


#### Auxiliary network devices selectors
These selectors are applicable when "deviceType" is "auxNetDevice".

|     Field     | Required |                                                              Description                                                               |                  Type/Defaults                   |                                     Example/Accepted values                                      |
|---------------|----------|----------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------|--------------------------------------------------------------------------------------------------|
| "vendors"     | N        | Target device's vendor Hex code as string                                                                                              | `string` list Default: `null`                    | "vendors": ["8086", "15b3"]                                                                      |
| "devices"     | N        | Parent PCI device Hex code as string                                                                                                   | `string` list Default: `null`                    | "devices": ["154c", "1889", "1018"]                                                              |
| "drivers"     | N        | Target device driver names as string                                                                                                   | `string` list Default: `null`                    | "drivers": ["vfio-pci"]                                                                          |
| "pfNames"     | N        | functions from PF matches list of PF names                                                                                             | `string` list Default: `null`                    | "pfNames": ["enp2s2f0"] (See follow-up sections for some advance usage of "pfNames")             |
| "rootDevices" | N        | functions from PF matches list of PF PCI addresses                                                                                     | `string` list Default: `null`                    | "rootDevices": ["0000:86:00.0"] (See follow-up sections for some advance usage of "rootDevices") |
| "linkTypes"   | N        | The link type of the net device associated with the PCI device                                                                         | `string` list Default: `null`                    | "linkTypes": ["ether"]                                                                           |
| "isRdma"      | N        | Mount RDMA resources. Incompatible with vdpaType                                                                                       | `bool` values `true` or `false` Default: `false` | "isRdma": `true`                                                                                 |
| "auxTypes"    | N        | List of vendor specific auxiliary network device types. Device type can be determined by its name: <driver_name>.<kind_of_a_type>.<id> | `string` list Default: `null`                    | "auxTypes": ["sf", "eth"]                                                                        |

[//]: # (The tables above generated using: https://ozh.github.io/ascii-tables/)

#### AdditionalInfo field

This field defines a method to add information as part of the environment variable the sriov-network-device-plugin injects to the container.

Examples:
```
"additionalInfo": {
   "*": {
     "token": "3e49019f-412f-4f02-824e-4cd195944205"
   }
}     
```
The '*' is a special key telling the device plugin to inject this info to all the devices matching the selector.
you can also specify a specific pci address if you want to override or add more information to a specific device

```
"additionalInfo": {
   "*": {
     "token": "3e49019f-412f-4f02-824e-4cd195944205"
   },
    "0000:86:00.0": {
    "additional-token": "6e7dc135-7a1c-456f-a008-c2cfb37997be"
    }

}   

output for pod allocating a different deviceID:
{"0000:3b:02.6":{"extraInfo":{"token":"3e49019f-412f-4f02-824e-4cd195944205"}

output for pod allocating the specific deviceID:
{"0000:86:00.0":{"extraInfo":{"token":"3e49019f-412f-4f02-824e-4cd195944205","additional-token": "6e7dc135-7a1c-456f-a008-c2cfb37997be"}
```

When having the same key for both the "*" and spefic deviceID variable the specific one will be taken

```
"additionalInfo": {
   "*": {
     "token": "original"
   },
    "0000:86:00.0": {
    "token": "specific"
    }

}   

output for pod allocating a different deviceID:
{"0000:3b:02.6":{"extraInfo":{"token":"original"}

output for pod allocating the specific deviceID:
{"0000:86:00.0":{"extraInfo":{"token":"specific"}
```

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

This plugin does not bind or unbind any driver to any device whether it's PFs or VFs. It also doesn't create virtual functions either. Usually, the virtual functions are created at boot time when kernel module for the device is loaded. Same with SFs. Required device drivers could be loaded on system boot-up time by allow-listing/deny-listing the right modules. But plugin needs to be aware of the driver type of the resources (i.e. devices) that it is registering as K8s extended resource so that it's able to create appropriate Device Specs for the requested resource.

For example, if the driver type is uio (i.e. igb_uio.ko) then there are specific device files to add in Device Spec. For vfio-pci, device files are different. And if it is Linux kernel network driver then there is no device file to be added.

The idea here is, user creates a resource config for each resource pool as shown in [Config parameters](#config-parameters) by specifying the resource name and a "selector object" or list of "selector objects". Each "selector object" contains "selector(s)".

The device plugin will initially discover all PCI network resources in the host and populate an initial "device list". If device type is Auxiliary network device (auxNetDevice), then for each discovered PCI device of type Netdevice the plugin discovers auxiliary devices. Each "resource pool" then applies its selector object(s) in order to the list of discovered devices. The plugin will add devices that satisfy the selector object's constraints to the resource pool. Each "selector" specified in the selector object narrows down the list of devices for the resource pool. Currently, the selectors are applied in following order:

1. "vendors"      - The vendor hex code of device
2. "devices"      - The device hex code of device
3. "drivers"      - The driver name the device is registered with
4. "pciAddresses" - The pci address of the device in BDF notation (if device type is accelerator or netDevice)
5. "acpiIndexes"  - The acpi index of the device
6. "auxTypes"     - The type of auxiliary network device (if device type is auxNetDevice).
7. "pfNames"      - The Physical function name
8. "rootDevices"  - The Physical function PCI address
9. "linkTypes"    - The link type of the net device associated with the PCI device

If a single device matches multiple selector objects, it will only be allocated to the first one.

The "pfNames" and "rootDevices" selectors can be used to specify a list and/or range of VFs/SFs for a pool in the below format
````
"<PFName>#<SingleFunc>,<FirstFunc>-<LastFunc>,<SingleFunc>,<SingleFunc>,<FirstFunc>-<LastFunc>"
````

Or

````json
"<RootDevice>#<SingleFunc>,<FirstFunc>-<LastFunc>,<SingleFunc>,<SingleFunc>,<FirstFunc>-<LastFunc>"
````

Where:

    `<PFName>`     - is the PF interface name
    `<RootDevice>` - is the PF PCI address
    `<SingleFunc>` - is a single function index (0-based) that is included into the pool
    `<FirstFunc>`  - is the first function index (0-based) that is included into the range
    `<LastFunc>`   - is the last function index (0-based) that is included into the range

Example:

The selector for interface named `netpf0` and VF 0, 2 up to 7 (included 2 and 7) and 9 will look like:

````json
"pfNames": ["netpf0#0,2-7,9"]
````

The selector for PCI address `0000:86:00.0` and VF 0, 1, 3, 4 will look like:

````json
"rootDevices": ["0000:86:00.0#0-1,3,4"]
````

If only PF network interface or PF PCI address is specified in the selector, then assuming that all VFs or SFs of this interface are going to the pool.

### Workflow

- Load device's (Physical function if it is SR-IOV capable) kernel module and bind the driver to the PF
- Create required VFs or SFs
- Bind all VFs with right drivers
- Create a resource config map
- Run SR-IOV Network Device Plugin (as daemonset)

On successful run, the allocatable resource list for the node should be updated with resource discovered by the plugin as shown below. Note that the resource name is appended with the `-resource-prefix` i.e. `"intel.com/sriov_net_A"`.

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

The [images](./images) directory contains example Dockerfile, sample specs along with build scripts to deploy the SR-IOV Network Device Plugin as daemonset. Please see [README.md](./images/README.md) for more information about the Docker images.

````sh
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

There are some example Pod specs and related network CRD yaml files in [deployments](./deployments) directory for a sample deployment with Multus.

Leave the SR-IOV Network Device Plugin running and open a new terminal session for following steps.

#### Deploy test Pod connecting to pre-created SR-IOV network

````sh
$ kubectl create -f pod-tc1.yaml
pod "testpod1" created

$ kubectl get pods
NAME                  READY     STATUS    RESTARTS   AGE
testpod1              1/1       Running   0          3s
````

#### Verify Pod network interfaces

````sh
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

````sh
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

````sh
[cloudadmin@controller-1 ~]$ kubectl get dnet -n example-sriov
NAME         AGE
management   6s
sriov-a      14m
sriov-b      13m
````

#### Connect your networks to existing SR-IOV Device Pools

The Spec.Options.device_pool mandatory parameter denotes the Device Pool used by the network.
Make sure this parameter is set to the name(s) of your existing SR-IOV Device Pool(s)!

````sh
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

````sh
[cloudadmin@controller-1 ~]$ grep -B1 sriov_ sriov_pod.yaml
      requests:
        nokia.k8s.io/sriov_ens2f1: '2'
      limits:
        nokia.k8s.io/sriov_ens2f1: '2'
````

Then instantiate the Pod:

````sh
[cloudadmin@controller-1 ~]$ kubectl create -f sriov_pod.yaml
pod/sriov-pod created
````

#### Verify status and the network connections of the demo Pod

````sh
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

The allocated device information is exported in Container's environment variable. The variable name is `PCIDEVICE_` appended with full extended resource name (e.g. intel.com/sriov etc.) which is capitailzed and any special characters (".", "/") are replaced with underscore ("_"). In case of multiple devices from same extended resource pool, the device IDs are delimited with commas (",").

For example, if 2 devices are allocated from `intel.com/sriov` extended resource then the allocated device information will be found in following env variable:
`PCIDEVICE_INTEL_COM_SRIOV=0000:03:02.1,0000:03:04.3`

There is also another environment variable that expose in json format more information about the device that was allocated like mount and addition variables if requested.
The variable name is `PCIDEVICE_<RESOUCE_NAME>_INFO` appended with full extended resource name (e.g. intel.com/sriov etc.) which is capitailzed and any special characters (".", "/") are replaced with underscore ("_")

vfio device example:
```
PCIDEVICE_INTEL_COM_DPDK_NIC_1_INFO={"0000:3b:02.6":{"vfio":{"vfio-dev-mount":"/dev/vfio/169","vfio-mount":"/dev/vfio/vfio"},"vhost":{"net-mount":"/dev/vhost-net","tun-mount":"/dev/net/tun"}}}

# Adding additionalInfo field to the resource definition will add additional information for every device allocated. e.g:
PCIDEVICE_INTEL_COM_DPDK_NIC_1_INFO={"0000:3b:02.6":{"extraInfo":{"token":"3e49019f-412f-4f02-824e-4cd195944205"},"vfio":{"vfio-dev-mount":"/dev/vfio/169","vfio-mount":"/dev/vfio/vfio"},"vhost":{"net-mount":"/dev/vhost-net","tun-mount":"/dev/net/tun"}}}
```

## Virtual Deployments Support

### Configure Device Plugin extended selectors in virtual environments

SR-IOV Network Device Plugin supports running in a virtualized environment.  However, not all device selectors are
applicable as the VFs are passthrough to the VM without any association to their respective PF, hence any device
selector that relies on the association between a VF and its PF will not work and therefore the _pfNames_ and
_rootDevices_ extended selectors will not work in a virtual deployment.  The common selector _pciAddress_ can be
used to select the virtual device.

## CNI plugins in virtual environments

SR-IOV CNI plugin doesn't support running in a virtualized environment since it always requires accessing to PF device
which usually doesn't exist in VM. The recommended CNI plugin to use in a virtualized environment is the [host-device](https://www.cni.dev/plugins/current/main/host-device/) CNI plugin.

### Virtual environments with no iommu

SR-IOV Network Device Plugin supports allocating VFIO devices in a virtualized environment without a virtualized iommu.
For more information refer to [this](./docs/dpdk/README-virt.md).

## Multi Architecture Support

The supported architectures:

- AMD64
- ARM64
- PPC64LE

Buiding image for AMD64:

```sh
$ DOCKERFILE=Dockerfile make image
```

Building image for PPC64LE:

```sh
$ DOCKERFILE=images/Dockerfile.ppc64le TAG=ghcr.io/k8snetworkplumbingwg/sriov-device-plugin:latest-ppc64le make image
```

## Container Device Interface
To enable Container Device Interface (CDI) deployment please the see [CDI](deployments/cdi/README.md).

## Issues and Contributing

We welcome your feedback and contributions to this project. Please see the [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.
