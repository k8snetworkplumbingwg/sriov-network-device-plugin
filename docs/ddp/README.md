

# SR-IOV network device plugin with DDP
Dynamic Device Personalization aka DDP allows dynamic reconfiguration of the packet processing pipeline of Intel Ethernet 700 Series to meet specific use case needs on demand, adding new packet processing pipeline configuration *profiles* to a network adapter at run time, without resetting or rebooting the server.

(ref: [Dynamic Device Personalization for Intel® Ethernet 700 Series](https://software.intel.com/en-us/articles/dynamic-device-personalization-for-intel-ethernet-700-series))

The SR-IOV network device plugin could be used to identify currently running DDP *profiles*, thus allow it to filter Virtual Functions by their DDP profile names.

The Intel Ethernet 700 Series, a DDP profiles can be loaded/unloaded into the NIC using i40e kernel module(v2.7.26+) and ethtool OR using DPDK i40e pollmode driver & DPDK api. In this documentation we will cover i40e Kernel driver mode.

## Pre-requisites
 * Firmware: v6.01 or newer
 * Driver: i40e v2.7.26 or newer

Refer to [Intel downloadcenter](https://downloadcenter.intel.com/) for latest firmware and drivers for Intel Ethernet 700 Series.

You can use `ethtool` to get driver and firmware information of the controller.

```
# ethtool -i enp2s0f0
driver: i40e
version: 2.9.21
firmware-version: 7.00 0x80004cda 1.2154.0
expansion-rom-version:
bus-info: 0000:02:00.0
supports-statistics: yes
supports-test: yes
supports-eeprom-access: yes
supports-register-dump: yes
supports-priv-flags: yes
```

## 1. Install DDP profiles
On each node, download and extract desired DDP profiles into `/lib/firmware/intel/i40e/ddp/` directory.

The latest list of available packages can be found [here]( https://downloadcenter.intel.com/search?keyword=Dynamic+Device+Personalization).

For example, to download and extract the GTP profile:

```
$ wget https://downloadmirror.intel.com/27587/eng/gtp.zip
$ unzip gtp.zip
$ mkdir -pv /lib/firmware/intel/i40e/ddp
$ cp gtp.pkgo /lib/firmware/intel/i40e/ddp/
```

## 2. Load a DDP profile in to the NIC

Use Linux `ethtool` utility to load a DDP profile into the controller.
> Note: You can only load DDP profile into a controller using only first Physical Function(PF0).
```
$ ethtool -f enp2s0f0 gtp.pkgo 100
```
## 3. Create SR-IOV Virtual Functions

Create desired number of VFs using PF interfaces of the controller.

```
$ echo 2 > /sys/class/net/enp2s0f0/device/sriov_numvfs

```

## 4. Verify that correct profile is loaded
You can use another Linux utility for Intel 700 Series called `ddptool` to query current DDP profile information. This tool can downloaded from [here]( https://downloads.sourceforge.net/project/e1000/ddptool%20stable/ddptool-1.0.0.0/ddptool-1.0.0.0.tar.gz).

```
[root@silpixa00396659 ~]# ddptool -a
Intel(R) Dynamic Device Personalization Tool
DDPTool version 1.0.0.0
Copyright (C) 2019 Intel Corporation.

NIC  DevId D:B:S.F      DevName         TrackId  Version      Name
==== ===== ============ =============== ======== ============ ==============================
001) 1572  0000:02:00.0 enp2s0f0        80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
002) 1572  0000:02:00.1 enp2s0f1        80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
003) 1572  0000:02:00.2 enp2s0f2        80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
004) 1572  0000:02:00.3 enp2s0f3        80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
005) 154C  0000:03:02.0 N/A             80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
006) 154C  0000:03:02.1 enp3s2f1        80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
007) 154C  0000:03:02.2 enp3s2f2        80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
008) 154C  0000:03:02.3 N/A             80000008 1.0.3.0      GTPv1-C/U IPv4/IPv6 payload
```

Take note of the `Name` of the profile from ddptool output. This name will be used in `"ddpProfiles"` selector in plugin's resource pool configurations.

## 5. Create resource config with DDP profile selector

Create ConfigMap for device plugin:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: sriovdp-config
  namespace: kube-system
data:
  config.json: |
    {
        "resourceList": [{
                "resourceName": "x700_gtp",
                "selectors": {
                    "vendors": ["8086"],
                    "devices": ["154c"],
                    "ddpProfiles": ["GTPv1-C/U IPv4/IPv6 payload"]
                }
            },
            {
                "resourceName": "x700_pppoe",
                "selectors": {
                    "vendors": ["8086"],
                    "devices": ["154c"],
                    "ddpProfiles": ["E710 PPPoE and PPPoL2TPv2"]
                }
            }
        ]
    }

```

```
$ kubectl create -f configMap.yaml
```

## 6. Deploy SR-IOV network device plugin
Once the ConfigMap for the device plugin is created/updated you can deploy the SR-IOV network device plugin as [usual](https://github.com/intel/sriov-network-device-plugin#example-deployments). When everything is good, we should see that device plugin is able to discover VFs with DDP profile names given in the resource pool selector.

```
[root@silpixa00396659 ~]# kubectl get node node1 -o json | jq ".status.allocatable"
{
  "cpu": "8",
  "ephemeral-storage": "169986638772",
  "hugepages-1Gi": "0",
  "hugepages-2Mi": "8Gi",
  "intel.com/x700_gtp": "2",
  "intel.com/x700_pppoe": "0",
  "memory": "7880620Ki",
  "pods": "100"
}

```

## 7. Create net-attach-def CRs
This deployment is using Multus and SR-IOV CNI plugin for Pod network attachment of SR-IOV VFs. For this, we need to create a new net-attach-def CR that reference to the new resource pool.

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

## 8. Deploy workloads
Once we can verify that VFs are registered by the device plugin under correct resource pool with specific DDP profile, we can request those VFs from our workload as normal.

```
$ kubectl create -f pod-gtp.yaml
```

The sample ConfigMap and Pod specs are available in this directory.

## References
* https://software.intel.com/en-us/articles/dynamic-device-personalization-for-intel-ethernet-700-series
* https://www.intel.com/content/www/us/en/architecture-and-technology/ethernet/dynamic-device-personalization-brief.html
* https://builders.intel.com/docs/networkbuilders/implementing-a-high-performance-bng-with-intel-universal-nfvi-packet-forwarding-platform-technology.pdf
