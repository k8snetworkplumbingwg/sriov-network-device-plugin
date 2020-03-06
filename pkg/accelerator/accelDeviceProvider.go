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
	"strconv"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/jaypipes/ghw"
)

type accelDeviceProvider struct {
	deviceList []types.AccelDevice
	rFactory   types.ResourceFactory
}

// NewAccelDeviceProvider DeviceProvider implementation from accelDeviceProvider instance
func NewAccelDeviceProvider(rf types.ResourceFactory) types.DeviceProvider {
	return &accelDeviceProvider{
		rFactory:   rf,
		deviceList: make([]types.AccelDevice, 0),
	}
}

func (ap *accelDeviceProvider) GetDevices() []types.PciDevice {
	newPciDevices := make([]types.PciDevice, len(ap.deviceList))
	for i := range ap.deviceList {
		newPciDevices[i] = ap.deviceList[i]
	}
	return newPciDevices
}

func (ap *accelDeviceProvider) AddTargetDevices(devices []*ghw.PCIDevice, deviceCode int) error {

	for _, device := range devices {
		devClass, err := strconv.ParseInt(device.Class.ID, 16, 64)
		if err != nil {
			glog.Warningf("accelerator AddTargetDevices(): unable to parse device class for device %+v %q", device, err)
			continue
		}

		if devClass == int64(deviceCode) {
			vendor := device.Vendor
			vendorName := vendor.Name
			if len(vendor.Name) > 20 {
				vendorName = string([]byte(vendorName)[0:17]) + "..."
			}
			product := device.Product
			productName := product.Name
			if len(product.Name) > 40 {
				productName = string([]byte(productName)[0:37]) + "..."
			}
			glog.Infof("accelerator AddTargetDevices(): device found: %-12s\t%-12s\t%-20s\t%-40s", device.Address, device.Class.ID, vendorName, productName)

			if newDevice, err := NewAccelDevice(device, ap.rFactory); err == nil {
				ap.deviceList = append(ap.deviceList, newDevice)
			} else {
				glog.Errorf("accelerator AddTargetDevices() error adding new device: %q", err)
			}

		}
	}
	return nil
}
