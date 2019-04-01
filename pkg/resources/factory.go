// Copyright 2018 Intel Corp. All Rights Reserved.
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

package resources

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type resourceFactory struct {
	endPointPrefix string
	endPointSuffix string
}

var instance *resourceFactory

// NewResourceFactory returns an instance of Resource Server factory
func NewResourceFactory(prefix, suffix string) types.ResourceFactory {

	if instance == nil {
		return &resourceFactory{
			endPointPrefix: prefix,
			endPointSuffix: suffix,
		}
	}
	return instance
}

// GetResourceServer returns an instance of ResourceServer for a ResourcePool
func (rf *resourceFactory) GetResourceServer(rp types.ResourcePool) (types.ResourceServer, error) {
	if rp != nil {
		return newResourceServer(rf.endPointPrefix, rf.endPointSuffix, rp), nil
	}
	return nil, fmt.Errorf("factory: unable to get resource pool object")
}

// GetInfoProvider returns an instance of DeviceInfoProvider using name as string
func (rf *resourceFactory) GetInfoProvider(name string) types.DeviceInfoProvider {
	switch name {
	case "vfio-pci":
		return newVfioResourcePool()
	case "uio":
		return newUioResourcePool()
	default:
		return newNetDevicePool()
	}
}

// GetSelector returns an instance of DeviceSelector using selector attribute string and its associated values
func (rf *resourceFactory) GetSelector(attr string, values []string) (types.DeviceSelector, error) {
	// glog.Infof("GetSelector(): selector for attribute: %s", attr)
	switch attr {
	case "vendors":
		return newVendorSelector(values), nil
	case "devices":
		return newDeviceSelector(values), nil
	case "drivers":
		return newDriverSelector(values), nil
	default:
		return nil, fmt.Errorf("GetSelector(): invalid attribute %s", attr)
	}
}

// GetResourcePool returns an instance of resourcePool
func (rf *resourceFactory) GetResourcePool(rc *types.ResourceConfig, deviceList []types.PciNetDevice) (types.ResourcePool, error) {
	filteredDevices := make([]types.PciNetDevice, 0)
	filteredDevicesSet := make(map[types.PciNetDevice]bool)
	for _, selector := range rc.Selectors {
		selectedDevices := deviceList
		// filter by vendor  list
		if selector.Vendors != nil && len(selector.Vendors) > 0 {
			if vendorsSelector, err := rf.GetSelector("vendors", selector.Vendors); err == nil {
				selectedDevices = vendorsSelector.Filter(selectedDevices)
			}
		}
		// filter by device list
		if selector.Devices != nil && len(selector.Devices) > 0 {
			if devicesSelector, err := rf.GetSelector("devices", selector.Devices); err == nil {
				selectedDevices = devicesSelector.Filter(selectedDevices)
			}
		}
		// filter by driver list
		if selector.Drivers != nil && len(selector.Drivers) > 0 {
			if driversSelector, err := rf.GetSelector("drivers", selector.Drivers); err == nil {
				selectedDevices = driversSelector.Filter(selectedDevices)
			}
		}

		for _, dev := range selectedDevices {
			if !filteredDevicesSet[dev] {
				filteredDevices = append(filteredDevices, dev)
				filteredDevicesSet[dev] = true
			}
		}
	}

	devicePool := make(map[string]types.PciNetDevice, 0)
	apiDevices := make(map[string]*pluginapi.Device)
	for _, dev := range filteredDevices {
		pciAddr := dev.GetPciAddr()
		devicePool[pciAddr] = dev
		apiDevices[pciAddr] = dev.GetAPIDevice()
		glog.Infof("device added: [pciAddr: %s, vendor: %s, device: %s, driver: %s]",
			dev.GetPciAddr(),
			dev.GetVendor(),
			dev.GetDeviceCode(),
			dev.GetDriver())
	}

	rPool := newResourcePool(rc, apiDevices, devicePool)
	return rPool, nil
}
