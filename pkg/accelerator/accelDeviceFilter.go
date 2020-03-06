// Copyright 2020 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package accelerator

import (
	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

type accelDeviceFilter struct {
	types.AccelDeviceSelectors
	rFactory types.ResourceFactory
	isRdma   bool
}

// NewAccelDeviceFilter instantiates accelDeviceFilter
func NewAccelDeviceFilter(selectors *types.AccelDeviceSelectors, rf types.ResourceFactory) types.DeviceFilter {
	return &accelDeviceFilter{
		AccelDeviceSelectors: *selectors,
		rFactory:             rf,
	}
}

func (nf *accelDeviceFilter) GetFilteredDevices(devices []types.PciDevice) []types.PciDevice {
	filteredDevice := devices

	rf := nf.rFactory
	// filter by vendor list
	if nf.Vendors != nil && len(nf.Vendors) > 0 {
		if selector, err := rf.GetSelector("vendors", nf.Vendors); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by device list
	if nf.Devices != nil && len(nf.Devices) > 0 {
		if selector, err := rf.GetSelector("devices", nf.Devices); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by driver list
	if nf.Drivers != nil && len(nf.Drivers) > 0 {
		if selector, err := rf.GetSelector("drivers", nf.Drivers); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// convert to []AccelDevice to []PciDevice
	newDeviceList := make([]types.PciDevice, len(filteredDevice))
	for i, d := range filteredDevice {
		newDeviceList[i] = d
	}

	return newDeviceList
}
