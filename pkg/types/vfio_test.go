// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "testing"

func TestValidateVFIOConfig(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		need       bool
		selectors  []interface{}
		invalid    bool
	}{
		{name: "legacy default"},
		{name: "explicit legacy", mode: "legacy"},
		{name: "unknown mode", mode: "auto", invalid: true},
		{name: "iommu without native mode", need: true, invalid: true},
		{name: "native vfio", mode: "iommufd", selectors: []interface{}{&NetDeviceSelectors{DeviceSelectors: DeviceSelectors{Drivers: []string{VFIOPCIDriver}}}}},
		{name: "native no driver restriction", mode: "iommufd", selectors: []interface{}{&NetDeviceSelectors{}}, invalid: true},
		{name: "native mixed drivers", mode: "iommufd", selectors: []interface{}{&NetDeviceSelectors{DeviceSelectors: DeviceSelectors{Drivers: []string{VFIOPCIDriver, "mlx5_core"}}}}, invalid: true},
		{name: "native RDMA", mode: "iommufd", selectors: []interface{}{&NetDeviceSelectors{DeviceSelectors: DeviceSelectors{Drivers: []string{VFIOPCIDriver}}, GenericNetDeviceSelectors: GenericNetDeviceSelectors{IsRdma: true}}}, invalid: true},
		{name: "native vDPA", mode: "iommufd", selectors: []interface{}{&NetDeviceSelectors{DeviceSelectors: DeviceSelectors{Drivers: []string{VFIOPCIDriver}}, VdpaType: "vhost"}}, invalid: true},
		{name: "native no selectors", mode: "iommufd", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rc := ResourceConfig{VFIOMode: tc.mode, NeedIommu: tc.need, DeviceType: NetDeviceType, SelectorObjs: tc.selectors}
			err := rc.ValidateVFIOConfig()
			if (err != nil) != tc.invalid {
				t.Fatalf("validation error=%v, want invalid=%v", err, tc.invalid)
			}
		})
	}
}
