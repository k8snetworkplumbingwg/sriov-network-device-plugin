# Using UDEV Rules for changing NIC's network device name

To allow "pfNames" selector to consistently select a specific device across nodes it is required to define a fixed network device name. UDEV rules allow the user to change the NIC's network device name every time the node get rebooted:

1. Add a new rule:
```
# vi /etc/udev/rules.d/90-netnames.rules
SUBSYSTEM=="net", ACTION=="add", DRIVERS=="?*", KERNELS=="0000:02:00.0", NAME="pf0s0f1"
``` 

`SUBSYSTEM` Match against the subsystem of the device
`ACTION` Action to be done at the next reboot
`DRIVER`  Match against the name of the driver backing the device
`KERNELS` Match against the kernel name for the device, here the pci address
`NAME` The new name for the NIC

2. Run udev commands
```
# udevadm control --reload
# udevadm trigger --action=add -attr-match=subsystem=net
```

3. Reboot the machine or reload the driver for the NIC
