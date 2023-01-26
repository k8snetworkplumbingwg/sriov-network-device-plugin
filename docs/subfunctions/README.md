# Subfunctions

[Subfunction](https://docs.kernel.org/networking/devlink/devlink-port.html#subfunction) (SF) is a lightweight function that has a parent PCI function on which it is deployed. Subfunction is created and deployed in unit of 1. Unlike SRIOV VFs, a subfunction doesn’t require its own PCI virtual function. A subfunction communicates with the hardware through the parent PCI function.

## Usage

To use a subfunction, 3 steps setup sequence is followed:

1. create - create a subfunction;
2. configure - configure subfunction attributes;
3. deploy - deploy the subfunction;

Subfunction management is done using devlink port user interface. User performs setup on the subfunction management device.

### 1. Create
A subfunction is created using a devlink port interface. A user adds the subfunction by adding a devlink port of subfunction flavour. The devlink kernel code calls down to subfunction management driver (devlink ops) and asks it to create a subfunction devlink port. Driver then instantiates the subfunction port and any associated objects such as health reporters and representor netdevice.

### 2. Configure
A subfunction devlink port is created but it is not active yet. That means the entities are created on devlink side, the e-switch port representor is created, but the subfunction device itself is not created. A user might use e-switch port representor to do settings, putting it into bridge, adding TC rules, etc. A user might as well configure the hardware address (such as MAC address) of the subfunction while subfunction is inactive.

### 3. Deploy
Once a subfunction is configured, user must activate it to use it. Upon activation, subfunction management driver asks the subfunction management device to instantiate the subfunction device on particular PCI function. A subfunction device is created on [Auxiliary Bus](https://www.kernel.org/doc/html/latest/driver-api/auxiliary_bus.html). At this point a matching subfunction driver binds to the subfunction’s auxiliary device.


## Nvidia-Mellanox Scalable Functions

One of the implementation of Subfunctions is [Mellanox/ScalableFunctions](https://github.com/Mellanox/scalablefunctions/wiki).
Mellanox Scalable Function has its own function capabilities and its own resources. This means a Scalable Function has its own dedicated queues(txq, rxq, cq, eq). These queues are neither shared nor stolen from the parent PCI function. There is no special support needed from system BIOS to use Mellanox Scalable Functions. Scalable Functions do not require enabling PCI SR-IOV and co-exist with PCI SR-IOV Virtual Functions.

To utilize Scalable Functions as a resource following plugin configuration can be used:

For BlueField-2® NIC:

```json
{
    "resourceList": [
        {
            "resourceName": "bf2_sf",
            "resourcePrefix": "nvidia.com",
            "deviceType": "auxNetDevice",
            "selectors": {
                "vendors": ["15b3"],
                "devices": ["a2d6"],
                "pfNames": ["p0#1-5"],
                "auxTypes": ["sf"]
            }
        }
    ]
}
```

For ConnectX-6® Dx NIC for SFs and VFs:

```json
{
    "resourceList": [
        {
            "resourceName": "cx6dx_vf",
            "resourcePrefix": "nvidia.com",
            "selectors": {
                "vendors": ["15b3"],
                "devices": ["101e"],
            }
        },
        {
            "resourceName": "cx6dx_sf",
            "resourcePrefix": "nvidia.com",
            "deviceType": "auxNetDevice",
            "selectors": {
                "vendors": ["15b3"],
                "devices": ["101e"],
                "auxTypes": ["sf"]
            }
        }
    ]
}
```

On how to configure and use Scalable Functions please refer to [https://github.com/Mellanox/scalablefunctions/wiki](https://github.com/Mellanox/scalablefunctions/wiki)
