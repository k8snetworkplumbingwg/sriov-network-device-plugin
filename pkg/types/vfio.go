// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "fmt"

const (
	// VFIOPCIDriver is the driver required by native PCI VFIO.
	VFIOPCIDriver = "vfio-pci"
	// VFIOModeLegacy selects the VFIO group/container interface.
	VFIOModeLegacy = "legacy"
	// VFIOModeIOMMUFD selects the native VFIO cdev interface.
	VFIOModeIOMMUFD = "iommufd"
)

// ValidateVFIOConfig validates the VFIO options after selectors have been parsed.
func (rc *ResourceConfig) ValidateVFIOConfig() error {
	switch rc.VFIOMode {
	case "", VFIOModeLegacy:
		if rc.NeedIommu {
			return fmt.Errorf("needIommu requires vfioMode iommufd")
		}
		return nil
	case VFIOModeIOMMUFD:
	default:
		return fmt.Errorf("unsupported vfioMode %q", rc.VFIOMode)
	}
	if rc.DeviceType != NetDeviceType && rc.DeviceType != AcceleratorType {
		return fmt.Errorf("native VFIO requires a PCI resource pool")
	}
	if len(rc.SelectorObjs) == 0 {
		return fmt.Errorf("native VFIO requires vfio-pci selectors")
	}
	for _, selector := range rc.SelectorObjs {
		var drivers []string
		switch s := selector.(type) {
		case *NetDeviceSelectors:
			drivers = s.Drivers
			if s.IsRdma || s.VdpaType != "" {
				return fmt.Errorf("native VFIO cannot expose host RDMA or vDPA devices")
			}
		case *AccelDeviceSelectors:
			drivers = s.Drivers
		default:
			return fmt.Errorf("unsupported selector for native VFIO")
		}
		if len(drivers) != 1 || drivers[0] != VFIOPCIDriver {
			return fmt.Errorf("native VFIO selectors must specify only the vfio-pci driver")
		}
	}
	return nil
}
