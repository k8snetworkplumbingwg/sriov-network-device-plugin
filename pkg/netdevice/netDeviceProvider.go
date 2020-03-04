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
	"fmt"
	"strconv"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	"github.com/jaypipes/ghw"
	"github.com/vishvananda/netlink"
)

type netDeviceProvider struct {
	deviceList []types.PciNetDevice
	rFactory   types.ResourceFactory
}

// NewNetDeviceProvider DeviceProvider implementation from netDeviceProvider instance
func NewNetDeviceProvider(rf types.ResourceFactory) types.DeviceProvider {
	return &netDeviceProvider{
		rFactory:   rf,
		deviceList: make([]types.PciNetDevice, 0),
	}
}

func (np *netDeviceProvider) GetDevices() []types.PciDevice {
	newPciDevices := make([]types.PciDevice, len(np.deviceList))
	for i := range np.deviceList {
		newPciDevices[i] = np.deviceList[i]
	}
	return newPciDevices
}

func (np *netDeviceProvider) AddTargetDevices(devices []*ghw.PCIDevice, deviceCode int) error {

	for _, device := range devices {
		devClass, err := strconv.ParseInt(device.Class.ID, 16, 64)
		if err != nil {
			glog.Warningf("discoverDevices(): unable to parse device class for device %+v %q", device, err)
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
			glog.Infof("discoverDevices(): device found: %-12s\t%-12s\t%-20s\t%-40s", device.Address, device.Class.ID, vendorName, productName)

			// exclude device in-use in host
			if isDefaultRoute, _ := hasDefaultRoute(device.Address); !isDefaultRoute {

				aPF := utils.IsSriovPF(device.Address)

				if aPF && utils.SriovConfigured(device.Address) {
					// do not add this device in net device list
					continue
				}

				if newDevice, err := NewPciNetDevice(device, np.rFactory); err == nil {
					np.deviceList = append(np.deviceList, newDevice)
				} else {
					glog.Errorf("discoverDevices() error adding new device: %q", err)
				}

			}
		}
	}
	return nil
}

// hasDefaultRoute returns true if PCI network device is default route interface
func hasDefaultRoute(pciAddr string) (bool, error) {

	// inUse := false
	// Get net interface name
	ifNames, err := utils.GetNetNames(pciAddr)
	if err != nil {
		return false, fmt.Errorf("error trying get net device name for device %s", pciAddr)
	}

	if len(ifNames) > 0 { // there's at least one interface name found
		for _, ifName := range ifNames {
			link, err := netlink.LinkByName(ifName)
			if err != nil {
				glog.Errorf("expected to get valid host interface with name %s: %q", ifName, err)
			}

			routes, err := netlink.RouteList(link, netlink.FAMILY_V4) // IPv6 routes: all interface has at least one link local route entry
			for _, r := range routes {
				if r.Dst == nil {
					glog.Infof("excluding interface %s:  default route found: %+v", ifName, r)
					return true, nil
				}
			}
		}
	}

	return false, nil
}
