# Running DPDK applications in Kubernetes

## Pre-requisites

### Hugepages
DPDK applications require Hugepages memory. Please refer to the [Hugepages section](http://doc.dpdk.org/guides/linux_gsg/sys_reqs.html#use-of-hugepages-in-the-linux-environment) in DPDK getting started guide on hugespages in DPDK.

Kubernetes nodes can only advertise a single size pre-allocated hugepages. Which means even though one can have both 2M and 1G hugepages in a system, Kubernetes will only recognize the default hugepages as schedulable resources. Workload can request for hugepages using resource requests and limits specifying `hugepage-2Mi` or `hugepage-1Gi` resource references.

> One important thing to note here is that when requesting hugepage resources, either memory or CPU resource requests need to be specified.

For more information on hugepage support in Kubernetes please see [here](https://kubernetes.io/docs/tasks/manage-hugepages/scheduling-hugepages/).


### VF drivers
DPDK applications require devices to be attached with supported dpdk backend driver.
* For Intel® x700 series NICs `igb_uio` or `vfio-pci` is required.
* For Mellanox ConnectX®-4 Lx, ConnectX®-5 Adapters `mlx5_core` or `mlx5_ib` is required.

### Privileges 
Certain privileges are required for dpdk application to function properly in Kubernetes Pod. The level of privileges depend on the application and the host device driver attached.

For example, devices with `igb_uio` driver requires a Pod to run with full privilege. With `vfio-pci` an application can run in a non-privilege Pod with only **IPC_LOCK** capability added.


# Example deployment
This directory includes sample deployment yaml files showing how to deploy a dpdk application in Kubernetes with in non-privileged Pod with SR-IOV VF attached to vfio-pci driver.

## Prepare host for vfio support and hugepages memory

1. On CentOS 7, edit `/etc/default/grub` file and add the following kernel boot parameters to enable iommu and create 8GB of 2M size hugepages.

```
GRUB_CMDLINE_LINUX="crashkernel=auto nomodeset rhgb quiet iommu=pt intel_iommu=on default_hugepagesz=1G hugepagesz=1G hugepages=16 pci=realloc,assign-busses"
```

2. Rebuild grub.cfg
```
grub2-mkconfig -o /boot/grub2/grub.cfg
```

For UEFI boot, this cmd will be grub2-mkconfig -o /boot/efi/EFI/<distro-name>/grub.cfg.
For example, on CentOS: 
```
grub2-mkconfig -o /boot/efi/EFI/centos/grub.cfg
```

3. Reboot

4. Confirm that the system started with above parameter
```
# cat /proc/cmdline
BOOT_IMAGE=/boot/vmlinuz-3.10.0-957.10.1.el7.x86_64 root=UUID=5b12f430-394c-4417-9064-7ab8091ff987 ro crashkernel=auto nomodeset rhgb quiet iommu=pt intel_iommu=on default_hugepagesz=1G hugepagesz=1G hugepages=16 pci=realloc,assign-busses

```
5. Confirm that Hugepages memory are allocated and mounted
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

6. Load vfio-pci module
```
modprobe vfio-pci
```

7. Create SR-IOV virtual functions and bind those VFs with vfio-pci driver. You can use or `driverctl` or [`dpdk-devbind.py`](https://github.com/DPDK/dpdk/blob/master/usertools/dpdk-devbind.py) to bind/unbind drivers using devices PCI addresses. Please see [here](https://dpdk-guide.gitlab.io/dpdk-guide/setup/binding.html) more information on NIC driver bindings.

# Performance
It is worth mentioning that to achieve maximum performance from a dpdk application the followings are required:

1. Application process needs to be pinned to some dedicated isolated CPUs. Detailing how to achieve this is out of scope of this document. You can refer to [CPU Manager for Kubernetes](https://github.com/intel/CPU-Manager-for-Kubernetes) that provides such functionality in Kubernetes

2. All application resources(CPUs, devices and memory) are from same NUMA locality
