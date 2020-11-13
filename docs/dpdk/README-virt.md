# Running DPDK applications in a Kubernetes virtual environment without virtualized iommu support

## Pre-requisites

In virtual deployments of Kubernetes where the underlying virtualization platform does not support a virtualized iommu, the VFIO driver needs to be loaded with a special 
flag.  The file **/etc/modprobe.d/vfio-noiommu.conf** must be created with the contents:

````
# cat /etc/modprobe.d/vfio-noiommu.conf
options vfio enable_unsafe_noiommu_mode=1
````

With the above option, vfio devices will be created with the form on the virtual host (VM):

````
/dev/vfio/noiommu-0
/dev/vfio/noiommu-1
...
````

The presence of noiommu-* devices will automatically be detected by the sriov-device-plugin.  The noiommu-N devices will be mounted **inside** the pod in their expected/normal location;

````
/dev/vfio/0
/dev/vfio/1
...
````
It should be noted that with no IOMMU, there is no way to ensure safe use of DMA.  When *enable_unsafe_noiommu_mode* is used, CAP_SYS_RAWIO privileges are necessary to work with groups and
containers using this mode.  

>Note: The most common use case for direct VF is with the **DPDK** framework which will require the use of privileged containers.

Use of this mode, specifically
binding a device without a native IOMMU group to a VFIO bus driver will taint the kernel.  Only no-iommu support for the vfio-pci bus is provided.  However, there are still those users
that want userspace drivers even under those conditions.

### Hugepages
DPDK applications require Hugepages memory. Please refer to the [Hugepages section](http://doc.dpdk.org/guides/linux_gsg/sys_reqs.html#use-of-hugepages-in-the-linux-environment) in DPDK getting started guide on hugespages in DPDK.

Make sure that the virtual environment is enabled for creating VMs with hugepage support.  

Kubernetes nodes can only advertise a single size pre-allocated hugepages. Which means even though one can have both 2M and 1G hugepages in a system, Kubernetes will only recognize the default hugepages as schedulable resources. Workload can request for hugepages using resource requests and limits specifying `hugepage-2Mi` or `hugepage-1Gi` resource references.

> One important thing to note here is that when requesting hugepage resources, either memory or CPU resource requests need to be specified.

For more information on hugepage support in Kubernetes please see [here](https://kubernetes.io/docs/tasks/manage-hugepages/scheduling-hugepages/).


### VF drivers
DPDK applications require devices to be attached with supported dpdk backend driver.
* For Intel® x700 series NICs `vfio-pci` is required.
* For Mellanox ConnectX®-4 Lx, ConnectX®-5 Adapters `mlx5_core` or `mlx5_ib` is required.

Native-bifurcating devices/drivers (i.e. Mellanox/mlx5_*) do not need to run with privilege.  Non-bifurcating devices/drivers (i.e. Intel/vfio-pci) the PODs need to run with privilege.  

### Privileges 
Certain privileges are required for dpdk application to function properly in Kubernetes Pod. The level of privileges depend on the application and the host device driver attached (as mentioned above).  When running in an environment without a fully virtualized IOMMU, the *enable_unsafe_noiommu_mode* of vfio must be used by creating a modprobe.d file.

````
# cat /etc/modprobe.d/vfio-noiommu.conf
options vfio enable_unsafe_noiommu_mode=1
````

With `vfio-pci` an application must run privilege Pod with  **IPC_LOCK** and **CAP_SYS_RAWIO** capability.

# Example deployment
This directory includes a sample deployment yaml files showing how to deploy a dpdk application in Kubernetes with a **privileged** Pod (_pod_testpmd_virt.yaml_). 

## Deploy Virtual machines with attached VFs

1. Depending on the virtualization environment, create a network that supports SR-IOV.  Configure the VF as per your requirements:
- Trusted On/Off
- Spoof-Checking On/Off

In a virtual environment, some VF characteristics are set by the underlying virtualization platform and are used 'as is' inside the VM.  A virtual deployment does not have access to the VFs associated PF.

2. Attach the VFs or associated ports to the VM

## Check that environment supports VFIO and hugepages memory

1.  After deployment of the VM, confirm that your hugepagesz parameter is present. 
````
sh-4.4# cat /proc/cmdline 
BOOT_IMAGE=(hd0,gpt1)/ostree/rhcos-92d66d9df4cafad87abd888fd1b22fd1d890e86bc2ad8b9009bb9faa4f403a95/vmlinuz-4.18.0-193.24.1.el8_2.dt1.x86_64 rhcos.root=crypt_rootfs random.trust_cpu=on console=tty0 console=ttyS0,115200n8 rd.luks.options=discard ostree=/ostree/boot.1/rhcos/92d66d9df4cafad87abd888fd1b22fd1d890e86bc2ad8b9009bb9faa4f403a95/0 ignition.platform.id=openstack nohz=on nosoftlockup skew_tick=1 intel_pstate=disable intel_iommu=on iommu=pt rcu_nocbs=2-3 tuned.non_isolcpus=00000003 default_hugepagesz=1G nmi_watchdog=0 audit=0 mce=off processor.max_cstate=1 idle=poll intel_idle.max_cstate=0
````
2. On the desired worker node, 

````
cat /proc/meminfo | grep -i hugepage
AnonHugePages:    245760 kB
ShmemHugePages:        0 kB
HugePages_Total:       8
HugePages_Free:        8
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:    1048576 kB
````
You should see your requested hugepage size and a non-zero HugePages_Total.

3. Confirm that Hugepages memory are allocated and mounted
```
# cat /proc/meminfo | grep -i hugepage
HugePages_Total:      16
HugePages_Free:       16
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:    1048576 kB

# mount | grep hugetlbfs
hugetlbfs on /dev/hugepages type hugetlbfs (rw,relatime)

```

5. Load vfio-pci module

````
# echo "options vfio enable_unsafe_noiommu_mode=1" > /etc/modprobe.d/vfio-noiommu.conf
````

```
modprobe vfio-pci
```

7. For non-bifurcating devices/drivers, bind the appropriate interfaces (VF) to the vfio-pci driver.  You can use or `driverctl` or [`dpdk-devbind.py`](https://github.com/DPDK/dpdk/blob/master/usertools/dpdk-devbind.py) to bind/unbind drivers using devices PCI addresses. Please see [here](https://dpdk-guide.gitlab.io/dpdk-guide/setup/binding.html) more information on NIC driver bindings.

Native-bifurcating devices/drivers can stay with the default binding.

# Performance
It is worth mentioning that to achieve maximum performance from a dpdk application the followings are required:

1. Application process needs to be pinned to some dedicated isolated CPUs. Detailing how to achieve this is out of scope of this document. You can refer to [CPU Manager for Kubernetes](https://github.com/intel/CPU-Manager-for-Kubernetes) that provides such functionality in Kubernetes.  In the virtualized case, cpu pinning and isolation must be considered at the phyiscal layer as well as the virtual layer.

2. All application resources(CPUs, devices and memory) are from same NUMA locality.  In the virtualized case, NUMA locality is controlled by the underlying virtualized platform for the VM.

# Usage

>When consuming a VFIO device in a virtual environment, a secondary network is not required as network configuration for the underlying VF should be performed at the hypervisor level.

An example of a noiommu deployment is shown in [pod_testpmd_virt.yaml](pod_testpmd_virt.yaml).  The configMap for the example is shown in [configMap-virt.yaml](configMap-virt.yaml).


