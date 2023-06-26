# SR-IOV Network Device Plugin with DDP
Dynamic Device Personalizationi aka DDP allows dynamic reconfiguration of the packet processing pipeline of Intel® Ethernet 800/700 series to meet specific use-case needs for on-demand, adding new packet processing pipeline configuration *packages* to a network adapter at run time, without resetting or rebooting the server.

The SR-IOV Network Device Plugin can be used to identify currently running DDP *packages*, allowing it to filter Virtual Functions (VFs) by their DDP package name.

In this documentation we will cover the kernel driver use-case only. DPDK configuration of DDP is out of scope.

## Dynamic Device Personalization for Intel® Ethernet Controller E810
For Intel® Ethernet Controller E810, a DDP package can be loaded into the NIC using the ice kernel driver. Current device DDP state can be determined with DDPTool or devlink.

### Recommended pre-requisites for SR-IOV Network Device Plugin
 * Firmware: v2.22 or newer
 * Driver: ice v1.2.1 or newer

### Additional tools for debug
 * DDP NIC state detection tool(s): devlink mainline kernel 5.10 or newer, DDPTool 1.0.1.4 or newer

## Dynamic Device Personalization for Intel® Ethernet Controller X710
For Intel® Ethernet Controller X710, a DDP package can be loaded into the NIC using i40e kernel driver and ethtool. Current device DDP state can be determined with DDPTool.

### Recommended pre-requisites for SR-IOV Network Device Plugin
 * Firmware: v8.30 or newer
 * Driver: i40e v2.7.26 or newer

### Additional tools for config/debug
 * Ethtool: RHEL* 7.5 or newer or Linux* kernel 4.0.1 or newer
 * DDP NIC state detection tool(s): DDPTool 1.0.0.0 or newer

## Short step by step configuration for E810 & X710 series NICs
### Install DDP packages
#### Intel® Ethernet Controller E810
Minimum verson of SRIOV Network Device Plugin needed is 3.3.2 to support DDP Profiles on E800 series NIC's. 

By default, ice driver will automatically load the default DDP package. If you require additional protocals beyond the default set available, download and extract the DDP package into your device firmware folder, typically `/lib/firmware/intel/ice/ddp`.
Additional packages to suit your use-case can be found at [Intel® download center](https://downloadcenter.intel.com/search?keyword=Dynamic+Device+Personalization)

#### Intel® Ethernet Controller X710
Download and extract the desired DDP packages into your device firmware folder, typically `/lib/firmware/intel/i40e/ddp/`.
Packages to suit your use-case can be found [Intel® download center](https://downloadcenter.intel.com/search?keyword=Dynamic+Device+Personalization)

### Load a DDP package
#### Intel® Ethernet Controller E810
##### DDP package applied to a single physical card
With E810, it is possible to load a different DDP package per physical card. Please see the ice driver readme for full details.
You must place the DDP package in your NIC firmware folder (typically ```/lib/firmware/updates/intel/ice/ddp/```), append physical card serial number to DDP package name and reload all the physical function drivers on that physical card.

##### DDP package applied to all physical cards on a host
Symbolically link your DDP package to the ```ice.pkg``` package in your NIC firmware folder (typically ```/lib/firmware/updates/intel/ice/ddp```/) and reload the ice driver.

#### Intel® Ethernet Controller X710
Please see the i40e driver readme for full details.
Use Linux `ethtool` utility to load a DDP package into the controller. No reload of the i40e driver required.
> Note: You can only load DDP package into a controller using only first Physical Function(PF0).
```
$ ethtool -f enp2s0f0 gtp.pkgo 100
```

### Create SR-IOV Virtual Functions

Create desired number of VFs using PF interfaces of the controllers.

```
$ echo 2 > /sys/class/net/${PF_NAME}/device/sriov_numvfs

```

### Verify that correct package is loaded
#### Intel® Ethernet Controller E810
Display the active DDP package with devlink starting with kernel ver. 5.10 or newer. See kernel documentation for more details [here](https://www.kernel.org/doc/html/latest/networking/devlink/ice.html).
You can also use another Linux utility for Intel® 800 Series called `ddptool` to query current DDP package information. This tool can be downloaded from sourceforge [here](https://sourceforge.net/projects/e1000/files/ddptool%20stable/) or GitHub [here](https://github.com/intel/ddp-tool).

#### Intel® Ethernet Controller X710
You can use Linux utility for Intel® 700 Series called `ddptool` to query current DDP package information. This tool can be downloaded from sourceforge [here](https://sourceforge.net/projects/e1000/files/ddptool%20stable/) or GitHub [here](https://github.com/intel/ddp-tool).

### Create resource config with DDP package selector

Create ConfigMap for SR-IOV Network Device Plugin:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: sriovdp-config
  namespace: kube-system
data:
  config.json: |
    {
        "resourceList": [
             {
                "resourceName": "e800_default",
                "selectors": [{
                    "vendors": ["8086"],
                    "devices": ["1889"],
                    "ddpProfiles": ["ICE OS Default Package"]
                }]
            },
            {
                "resourceName": "e800_comms",
                "selectors": [{
                    "vendors": ["8086"],
                    "devices": ["1889"],
                    "ddpProfiles": ["ICE COMMS Package"]
                }]
            },
            {
                "resourceName": "x700_gtp",
                "selectors": [{
                    "vendors": ["8086"],
                    "devices": ["154c"],
                    "ddpProfiles": ["GTPv1-C/U IPv4/IPv6 payload"]
                }]
            },
            {
                "resourceName": "x700_pppoe",
                "selectors": [{
                    "vendors": ["8086"],
                    "devices": ["154c"],
                    "ddpProfiles": ["E710 PPPoE and PPPoL2TPv2"]
                }]
            }
        ]
    }

```

```
$ kubectl create -f configMap.yaml
```

### Deploy SR-IOV Network Device Plugin
Once the ConfigMap for the device plugin is created/updated you can deploy the SR-IOV Network Device Plugin as [usual](https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin#example-deployments). When everything is good, we should see that device plugin is able to discover VFs with DDP package names given in the resource pool selector.

```
[root@localhost ~]# kubectl get node node1 -o json | jq ".status.allocatable"
{
  "cpu": "8",
  "ephemeral-storage": "169986638772",
  "hugepages-1Gi": "0",
  "hugepages-2Mi": "8Gi",
  "intel.com/x700_gtp": "2",
  "intel.com/x700_pppoe": "2",
  "intel.com/e800_default": "2",
  "intel.com/e800_comms": "2",
  "memory": "7880620Ki",
  "pods": "100"
}

```

### Create net-attach-def CRs
This deployment is using Multus and SR-IOV CNI plugin for Pod network attachment of SR-IOV VFs. For this, we need to create a new net-attach-def CR that references to the new resource pool.

```
$ cat crd-sriov-gtp.yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: sriov-net-gtp
  annotations:
    k8s.v1.cni.cncf.io/resourceName: intel.com/x700_gtp
spec:
  config: '{
    "cniVersion": "0.3.1",
    "type": "sriov",
    "name": "sriov-gtp",
    "ipam": {
      "type": "host-local",
      "subnet": "10.56.217.0/24",
      "routes": [{
        "dst": "0.0.0.0/0"
      }],
      "gateway": "10.56.217.1"
    }
}'
```

```
$ kubectl create -f crd-sriov-gtp.yaml
```

### Deploy workloads
Once we can verify that VFs are registered by the device plugin under correct resource pool with specific DDP package, we can request those VFs from our workload as normal.

```
$ kubectl create -f pod-gtp.yaml
```

The sample ConfigMap and Pod specs are available in this directory.

## References
* https://downloadcenter.intel.com/search?keyword=Dynamic+Device+Personalization
* https://github.com/intel/ddp-tool
* https://sourceforge.net/projects/e1000/files/ddptool%20stable/
* https://www.kernel.org/doc/html/latest/networking/devlink/ice.html
* https://www.intel.com/content/www/us/en/products/network-io/ethernet/controllers/ethernet-800-series-controllers.html
* https://software.intel.com/en-us/articles/dynamic-device-personalization-for-intel-ethernet-700-series
* https://www.intel.com/content/www/us/en/architecture-and-technology/ethernet/dynamic-device-personalization-brief.html
* https://builders.intel.com/docs/networkbuilders/implementing-a-high-performance-bng-with-intel-universal-nfvi-packet-forwarding-platform-technology.pdf
