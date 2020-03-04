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

package netdevice

import (
	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

type netDeviceFilter struct {
	types.NetDeviceSelectors
	rFactory types.ResourceFactory
	isRdma   bool
}

// NewNetDeviceFilter instantiates netDeviceFilter
func NewNetDeviceFilter(selectors *types.NetDeviceSelectors, rf types.ResourceFactory, isRdma bool) types.DeviceFilter {
	return &netDeviceFilter{
		NetDeviceSelectors: *selectors,
		rFactory:           rf,
		isRdma:             isRdma,
	}
}

func (nf *netDeviceFilter) GetFilteredDevices(devices []types.PciDevice) []types.PciDevice {
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

	// filter by PfNames list
	if nf.PfNames != nil && len(nf.PfNames) > 0 {
		if selector, err := rf.GetSelector("pfNames", nf.PfNames); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by linkTypes list
	if nf.LinkTypes != nil && len(nf.LinkTypes) > 0 {
		if len(nf.LinkTypes) > 1 {
			glog.Warningf("Link type selector should have a single value.")
		}
		if selector, err := rf.GetSelector("linkTypes", nf.LinkTypes); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by DDP Profiles list
	if nf.DDPProfiles != nil && len(nf.DDPProfiles) > 0 {
		if selector, err := rf.GetSelector("ddpProfiles", nf.DDPProfiles); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter for rdma devices
	if nf.isRdma {
		rdmaDevices := make([]types.PciDevice, 0)
		for _, dev := range filteredDevice {

			if dev.(types.PciNetDevice).GetRdmaSpec().IsRdma() {
				rdmaDevices = append(rdmaDevices, dev)
			}
		}
		filteredDevice = rdmaDevices
	}

	// convert to []PciNetDevice to []PciDevice
	newDeviceList := make([]types.PciDevice, len(filteredDevice))
	for i, d := range filteredDevice {
		newDeviceList[i] = d
	}

	return newDeviceList
}
