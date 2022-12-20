# Setting up Virtual Functions

The SR-IOV Network Device Plugin requires SR-IOV virtual functions (VFs) to be created on the host ahead of startup. The device plugin does not manage the creation of virtual functions, and it does not automatically update when there is a change in the virtual functions. Each time the number of VFs or the driver used by the VF change the SR-IOV Network Device Plugin needs to be restarted.
This guide gives an overview for setting up SR-IOV virtual functions on a linux host for various NICs.

* [Intel](#intel)
* [Mellanox](#mellanox)

## Intel

The below works on Intel SR-IOV enabled adapters including those from the 500, 700 and 800 Series Network Adapters. Note not all netowrk adapters support SR-IOV.

Cards from the following families can be enabled through the sysfs endpoint as described below:

* **Intel ice Driver**
  * Intel Ethernet E810 Network Adapters
  * Intel® Ethernet Network Adapter E810-CQDA2T
  * Intel® Ethernet Network Adapter E810-2CQDA2
  * Intel® Ethernet Network Adapter E810-CQDA2
  * Intel® Ethernet Network Adapter E810-CQDA1
  * Intel® Ethernet Network Adapter E810-CQDA1 for OCP
  * Intel® Ethernet Network Adapter E810-CQDA1 for OCP 3.0
  * Intel® Ethernet Network Adapter E810-CQDA2 for OCP 3.0
  * Intel® Ethernet Network Adapter E810-XXVDA4T
  * Intel® Ethernet Network Adapter E810-XXVDA4 for OCP 3.0
  * Intel® Ethernet Network Adapter E810-XXVDA2
  * Intel® Ethernet Network Adapter E810-XXVDA2 for OCP 3.0
  * Intel® Ethernet Network Adapter E810-XXVDA4

* **Intel i40e Driver**
  * Intel® Ethernet Network Adapter X722
  * Intel® Ethernet Network Adapter XXV710
  * Intel® Ethernet Converged Network Adapter XL710
  * Intel® Ethernet Network Adapter X710
  * Intel® Ethernet Converged Network Adapter X710

* **Intel ixgbe Driver**
  * Intel® 82599 10 Gigabit Ethernet Controller
  * Intel® Ethernet Converged Network Adapter X520 Series
  * Intel® Ethernet Server Adapter X520
  * Intel® Ethernet Controller X550
  * Intel® Ethernet Converged Network Adapter X550
  * Intel® Ethernet Controller X540
  * Intel® Ethernet Converged Network Adapter X540
  * Intel® Ethernet Connection X557

### Creating VFs with sysfs

First select a compatible NIC on which to create VFs and record its name (shown as PF_NAME below).

To create 8 virtual functions run:

```sh
echo 8 > /sys/class/net/${PF_NAME}/device/sriov_numvfs
```

To check that the VFs have been successfully created run:

```sh
lspci | grep "Virtual Function"
```

This method requires the creation of VFs each time the node resets. This can be handled automatically by placing the above command in a script that is run on startup such as `/etc/rc.local`.

### Common issues

#### **Cannot allocate memory**

Some cards using the `ixgbe` may require additional configuration. This is likely the case if the message `write error: Cannot allocate memory` is returned when creating VFs using the above method. This issue has been observed on the X552 NIC.

To resolve this issue, try setting the maximum allowed VFs in the driver config:

``` sh
modprobe -r ixgbe; modprobe ixgbe max_vfs=8
```

Next set the VFs as above

```sh
echo 8 > /sys/class/net/${PF_NAME}/device/sriov_numvfs
```

#### **Device or resource busy**

The message `write error: device or resource busy` in response to an attempt to create virtual functions can mean that some virtual functions have already been created. In order to change the number of VFs the number may first need to be set to zero:

```sh
echo 0 > /sys/class/net/${PF_NAME}/device/sriov_numvfs

echo 8 > /sys/class/net/${PF_NAME}/device/sriov_numvfs
```

## Mellanox

SRIOV-CNI support Mellanox ConnectX®-4 Lx and ConnectX®-5 adapter cards.
To enable SR-IOV functionality the following steps are required:

1- Enable SR-IOV in the NIC's Firmware.

> Installing Mellanox Management Tools (MFT) or mstflint is a pre-requisite, MFT can be downloaded from [here](http://www.mellanox.com/page/management_tools), mstflint package available in the various distros and can be downloaded from [here](https://github.com/Mellanox/mstflint).

Use Mellanox Firmware Tools package to enable and configure SR-IOV in firmware

```sh
# mst start
Starting MST (Mellanox Software Tools) driver set
Loading MST PCI module - Success
Loading MST PCI configuration module - Success
Create devices
```

Locate the HCA device on the desired PCI slot

```sh
# mst status
MST modules:
------------
    MST PCI module loaded
    MST PCI configuration module loaded
MST devices:
------------
/dev/mst/mt4115_pciconf0         - PCI configuration cycles access.
...
```

Enable SR-IOV

```sh
# mlxconfig -d /dev/mst/mt4115_pciconf0 q set SRIOV_EN=1 NUM_OF_VFS=8
...
Apply new Configuration? ? (y/n) [n] : y
Applying... Done!
-I- Please reboot machine to load new configurations.
```

Alternatively, use `mstconfig` from _mstflint_ package

```sh
# mstconfig -d 04:00.0 set SRIOV_EN=1 NUM_OF_VFS=8
...
Apply new Configuration ? (y/n) [n] : y
Applying... Done!
-I- Please reboot machine to load new configurations.
```

Where `04:00.0` is the NIC's PCI address.

Reboot the machine

```sh
# reboot
```

2- Enable SR-IOV in the NIC's Driver.

```sh
# ibdev2netdev
mlx5_0 port 1 ==> enp2s0f0 (Up)
mlx5_1 port 1 ==> enp2s0f1 (Up)

# echo 4 > /sys/class/net/enp2s0f0/device/sriov_numvfs
# ibdev2netdev -v
0000:02:00.0 mlx5_0 (MT4115 - MT1523X04353) CX456A - ConnectX-4 QSFP fw 12.23.1020 port 1 (ACTIVE) ==> enp2s0f0 (Up)
0000:02:00.1 mlx5_1 (MT4115 - MT1523X04353) CX456A - ConnectX-4 QSFP fw 12.23.1020 port 1 (ACTIVE) ==> enp2s0f1 (Up)
0000:02:00.5 mlx5_2 (MT4116 - NA)  fw 12.23.1020 port 1 (DOWN  ) ==> enp2s0f2 (Down)
0000:02:00.6 mlx5_3 (MT4116 - NA)  fw 12.23.1020 port 1 (DOWN  ) ==> enp2s0f3 (Down)
0000:02:00.7 mlx5_4 (MT4116 - NA)  fw 12.23.1020 port 1 (DOWN  ) ==> enp2s0f4 (Down)
0000:02:00.2 mlx5_5 (MT4116 - NA)  fw 12.23.1020 port 1 (DOWN  ) ==> enp2s0f5 (Down)

# lspci | grep Mellanox
02:00.0 Ethernet controller: Mellanox Technologies MT27700 Family [ConnectX-4]
02:00.1 Ethernet controller: Mellanox Technologies MT27700 Family [ConnectX-4]
02:00.2 Ethernet controller: Mellanox Technologies MT27700 Family [ConnectX-4 Virtual Function]
02:00.3 Ethernet controller: Mellanox Technologies MT27700 Family [ConnectX-4 Virtual Function]
02:00.4 Ethernet controller: Mellanox Technologies MT27700 Family [ConnectX-4 Virtual Function]
02:00.5 Ethernet controller: Mellanox Technologies MT27700 Family [ConnectX-4 Virtual Function]

# ip link show
...
enp2s0f2: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether c6:6d:7d:dd:2a:d5 brd ff:ff:ff:ff:ff:ff
enp2s0f3: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether 42:3e:07:68:da:fb brd ff:ff:ff:ff:ff:ff
enp2s0f4: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
    link/ether 42:68:f2:aa:c2:27 brd ff:ff:ff:ff:ff:ff
enp2s0f5: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT group default qlen 1000
...
```

To change the number of VFs reset the number to 0 then set the needed number

```sh
echo 0 > /sys/class/net/enp2s0f0/device/sriov_numvfs
```
