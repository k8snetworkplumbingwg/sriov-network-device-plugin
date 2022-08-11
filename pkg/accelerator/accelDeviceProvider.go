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
	"fmt"

	"github.com/golang/glog"
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type accelDeviceProvider struct {
	deviceList []*ghw.PCIDevice
	rFactory   types.ResourceFactory
}

// NewAccelDeviceProvider DeviceProvider implementation from accelDeviceProvider instance
func NewAccelDeviceProvider(rf types.ResourceFactory) types.DeviceProvider {
	return &accelDeviceProvider{
		rFactory:   rf,
		deviceList: make([]*ghw.PCIDevice, 0),
	}
}

func (ap *accelDeviceProvider) GetDiscoveredDevices() []*ghw.PCIDevice {
	return ap.deviceList
}

func (ap *accelDeviceProvider) GetDevices(rc *types.ResourceConfig) []types.HostDevice {
	newHostDevices := make([]types.HostDevice, 0)
	for _, device := range ap.deviceList {
		if newDevice, err := NewAccelDevice(device, ap.rFactory, rc); err == nil {
			newHostDevices = append(newHostDevices, newDevice)
		} else {
			glog.Errorf("accelerator GetDevices() error creating new device: %q", err)
		}
	}
	return newHostDevices
}

func (ap *accelDeviceProvider) AddTargetDevices(devices []*ghw.PCIDevice, deviceCode int) error {
	for _, device := range devices {
		devClass, err := utils.ParseDeviceID(device.Class.ID)
		if err != nil {
			glog.Warningf("accelerator AddTargetDevices(): unable to parse device class for device %+v %q", device, err)
			continue
		}

		if devClass == int64(deviceCode) {
			vendorName := utils.NormalizeVendorName(device.Vendor.Name)
			productName := utils.NormalizeProductName(device.Product.Name)
			glog.Infof("accelerator AddTargetDevices(): device found: %-12s\t%-12s\t%-20s\t%-40s", device.Address,
				device.Class.ID, vendorName, productName)

			ap.deviceList = append(ap.deviceList, device)
		}
	}
	return nil
}

func (ap *accelDeviceProvider) GetFilteredDevices(devices []types.HostDevice, rc *types.ResourceConfig) ([]types.HostDevice, error) {
	filteredDevice := devices
	af, ok := rc.SelectorObj.(*types.AccelDeviceSelectors)
	if !ok {
		return filteredDevice, fmt.Errorf("unable to convert SelectorObj to AccelDeviceSelectors")
	}

	rf := ap.rFactory
	// filter by vendor list
	if af.Vendors != nil && len(af.Vendors) > 0 {
		if selector, err := rf.GetSelector("vendors", af.Vendors); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by device list
	if af.Devices != nil && len(af.Devices) > 0 {
		if selector, err := rf.GetSelector("devices", af.Devices); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by driver list
	if af.Drivers != nil && len(af.Drivers) > 0 {
		if selector, err := rf.GetSelector("drivers", af.Drivers); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by pciAddresses list
	if af.PciAddresses != nil && len(af.PciAddresses) > 0 {
		if selector, err := rf.GetSelector("pciAddresses", af.PciAddresses); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	return filteredDevice, nil
}

func (ap *accelDeviceProvider) ValidConfig(rc *types.ResourceConfig) bool {
	_, ok := rc.SelectorObj.(*types.AccelDeviceSelectors)
	if !ok {
		glog.Errorf("unable to convert SelectorObj to AccelDeviceSelectors")
		return false
	}
	return true
}
