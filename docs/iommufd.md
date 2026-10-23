# Native VFIO access with IOMMUFD

A VFIO PCI function can expose two access interfaces. The legacy interface uses
`/dev/vfio/vfio` and `/dev/vfio/<group>`. The native interface uses
`/dev/vfio/devices/vfio<N>` and an IOMMUFD opened through `/dev/iommu`.
Both interfaces refer to the same PCI function. They are not separate resources.

Select one interface for each resource pool. The plugin returns only that
interface. It does not select an interface from the kernel version, hardware
model, or presence of a device node.

## Configuration

The following fields belong to the resource object, outside `selectors`:

| Field | Default | Effect |
| --- | --- | --- |
| `vfioMode` | `legacy` | Accepts `legacy` or `iommufd`. An omitted or empty value keeps legacy behavior. |
| `needIommu` | `false` | In `iommufd` mode, also exposes `/dev/iommu` to the allocated container. Invalid in legacy mode. |

Example for a VMM that runs inside the allocated container and opens its own
IOMMUFD:

```json
{
  "resourceList": [
    {
      "resourceName": "vfio_native",
      "resourcePrefix": "example.com",
      "vfioMode": "iommufd",
      "needIommu": true,
      "selectors": [
        {
          "drivers": ["vfio-pci"],
          "pciAddresses": ["0000:03:00.1", "0000:03:00.2"]
        }
      ]
    }
  ]
}
```

Replace the PCI addresses with the functions reserved for this pool. Native mode
supports PCI network and accelerator pools. Each selector must specify only the
`vfio-pci` driver. Host RDMA, vDPA, auxiliary devices, and no-IOMMU devices are
not supported in this mode.

If a host runtime opens the IOMMUFD outside the allocated container, leave
`needIommu` false. The runtime must have access to `/dev/iommu` and must bind the
allocated cdev to its IOMMUFD. False does not remove the IOMMUFD requirement.
The plugin does not create, pass, or share an IOMMUFD file descriptor.

Kata and KubeVirt deployments must select the mode supported by their complete
runtime stack. A cdev on the node does not prove that the runtime can use it.
An existing legacy deployment can keep its configuration unchanged.

## Access to host device nodes

Native mode checks device-node identity before allocation. The plugin container
must be able to inspect the host VFIO nodes. Add these entries to its DaemonSet:

```yaml
# Add to the device-plugin container. Retain its existing arguments and mounts.
args:
  - --host-dev-dir=/host-dev
volumeMounts:
  - name: host-vfio
    mountPath: /host-dev/vfio
    readOnly: true
# Add to the pod's volumes.
volumes:
  - name: host-vfio
    hostPath:
      path: /dev/vfio
      type: Directory
```

When a pool sets `needIommu: true`, also add:

```yaml
# Add to the container's volumeMounts.
- name: host-iommu
  mountPath: /host-dev/iommu
  readOnly: true
# Add to the pod's volumes.
- name: host-iommu
  hostPath:
    path: /dev/iommu
    type: CharDevice
```

These are inspection mounts. Allocation always returns paths under host `/dev`,
not `/host-dev`. The flag defaults to `/dev` for a plugin that already sees the
host nodes there. Keep the existing host sysfs access. The plugin uses
`/sys/bus/pci/devices/<BDF>/vfio-dev` and, when needed,
`/sys/class/misc/iommu/dev` to check device numbers. No new mount is required for
pools that use the default legacy mode.

## Allocation and failure behavior

The plugin checks that the selected PCI function is still bound to `vfio-pci`.
It resolves exactly one cdev from that function's sysfs entry. It checks that
the device node is a character device and that its major and minor numbers match
sysfs. With `needIommu`, it also checks the IOMMU control node.

A missing, ambiguous, or mismatched native node makes the device unhealthy.
Periodic health checks can restore its health after the node is repaired at the
same path. If rebinding changes the resolved cdev path, the device stays unhealthy.
Drain affected workloads and restart the plugin to discover the new mapping.
This rule keeps metadata and injected nodes consistent during an allocation.
Allocation checks the nodes again and returns an error on failure. It returns no
partial response and never falls back to the legacy interface. The checks do not
open a device or take DMA ownership. They cannot prove that a later runtime bind
will succeed, or prevent a device change after allocation.

For multiple native devices, direct allocation includes `/dev/iommu` once when
requested. The `PCIDEVICE_*_INFO` VFIO section contains `mode: iommufd`,
`cdev-mount`, and, when requested, `iommu-mount`. Native metadata contains no
legacy `mount` or `dev-mount` keys. Legacy metadata keeps its existing format.

## Coexistence and migration

Use separate resource names and disjoint PCI functions for legacy and native
pools on the same node. The plugin's existing pool selection prevents one PCI
function from appearing in more than one pool within one plugin instance. Do
not expose the same functions through independent plugin instances.

IOMMU groups remain a hardware isolation boundary. A cdev does not make devices
in a shared group independently assignable to unrelated guests. Keep each
allocated function isolated, or arrange ownership of the full group in the
runtime. This feature does not add group-aware scheduling.

Drain workloads before changing a pool's mode or rebinding its devices. A new
resource name is recommended when migrating consumers. Qualify the kernel,
container runtime, VMM, security policy, and application together before moving
production workloads. Platform family names alone do not determine support.

## Validation

Unit tests use temporary sysfs fixtures and harmless character-device metadata.
They cover configuration, cdev identity, cdev-only hosts, native provider
selection, control-node injection, metadata, allocation failure, and health
changes. They do not open VFIO or IOMMU devices.

Before deployment, test both an existing legacy workload and a native workload
on the intended host. Confirm that the native workload can bind the device,
map DMA, perform network I/O, and release the device. Repeat with missing nodes,
multiple allocated VFs, and the intended Kata or KubeVirt configuration.

See the [Linux VFIO cdev documentation](https://docs.kernel.org/driver-api/vfio.html#vfio-device-cdev)
and [QEMU IOMMUFD documentation](https://www.qemu.org/docs/master/devel/vfio-iommufd.html).
