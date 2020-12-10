# Using vDPA devices in Kubernetes
## Introduction to vDPA
vDPA (Virtio DataPath Acceleration) is a technology that enables the acceleration
of virtIO devices while allowing the implementations of such devices
(e.g: NIC vendors) to use their own control plane.

The consumers of the virtIO devices (VMs or containers) interact with the devices
using the standard virtIO datapath and virtio-compatible control paths (virtIO, vhost).
While the data-plane is mapped directly to the accelerator device, the control-plane
gets translated by the vDPA kernel framework.

The vDPA kernel framework is composed of the vDPA bus (/sys/bus/vdpa), vDPA devices
(/sys/bus/vdpa/devices) and vDPA drivers (/sys/bus/vdpa/drivers).
Currently, two vDPA drivers are implemented:
*  virtio_vdpa: Exposes the device as a virtio-net netdev
*  vhost_vdpa: Exposes the device as a vhost-vdpa device. This device uses an extension
of the vhost-net protocol to allow userspace applications access the rings directly

For more information about the vDPA framework, read the article on
[LWN.net](https://lwn.net/Articles/816063/) or the blog series written by one of the
main authors ([part 1](https://www.redhat.com/en/blog/vdpa-kernel-framework-part-1-vdpa-bus-abstracting-hardware),
[part 2](https://www.redhat.com/en/blog/vdpa-kernel-framework-part-2-vdpa-bus-drivers-kernel-subsystem-interactions),
[part3](https://www.redhat.com/en/blog/vdpa-kernel-framework-part-3-usage-vms-and-containers))

## vDPA Management
Currently, the management of vDPA devices is performed using the sysfs interface exposed
by the vDPA Framework. However, in order to decouple the management of vDPA devices from
the SR-IOV Device Plugin functionality, this low-level management is done in an external
library called [go-vdpa](https://github.com/k8snetworkplumbingwg/govdpa).

In the context of the SR-IOV Device Plugin and the SR-IOV CNI, the current plan is to
support only 1:1 mappings between SR-IOV VFs and vDPA devices despite the fact that
the vDPA Framework might support 1:N mappings.

Note that vDPA and RDMA are mutually exclusive modes.

## Tested NICs:
* Mellanox ConnectX®-6 DX

## Prerequisites
* Linux Kernel >= 5.12
* iproute >= 5.14

## vDPA device creation
Insert the vDPA kernel modules if not present:

    $ modprobe vdpa
    $ modprobe virtio-vdpa
    $ modprobe vhost-vdpa

Create a vDPA device using the `vdpa` management tool integrated into iproute2, e.g:

    $ vdpa mgmtdev show
    pci/0000:65:00.2:
      supported_classes net
    $ vdpa dev add name vdpa2 mgmtdev pci/0000:65:00.2
    $ vdpa dev list
    vdpa2: type network mgmtdev pci/0000:65:00.2 vendor_id 5555 max_vqs 16 max_vq_size 256

## Bind the desired vDPA driver
The vDPA bus works similar to the pci bus. To unbind a driver from a device, run:

    echo ${DEV_NAME} > /sys/bus/vdpa/devices/${DEV_NAME}/driver/unbind

To bind a driver to a device, run:

    echo ${DEV_NAME} > /sys/bus/vdpa/drivers/${DRIVER_NAME}/bind

## Configure the SR-IOV Device Plugin
See the sample [configMap](configMap.yaml) for an example of how to configure a vDPA device.

