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

	"github.com/golang/glog"
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type netDeviceProvider struct {
	deviceList []*ghw.PCIDevice
	rFactory   types.ResourceFactory
}

// NewNetDeviceProvider DeviceProvider implementation from netDeviceProvider instance
func NewNetDeviceProvider(rf types.ResourceFactory) types.DeviceProvider {
	return &netDeviceProvider{
		rFactory:   rf,
		deviceList: make([]*ghw.PCIDevice, 0),
	}
}

func (np *netDeviceProvider) GetDiscoveredDevices() []*ghw.PCIDevice {
	return np.deviceList
}

func (np *netDeviceProvider) GetDevices(rc *types.ResourceConfig, selectorIndex int) []types.HostDevice {
	newHostDevices := make([]types.HostDevice, 0)
	for _, device := range np.deviceList {
		if newDevice, err := NewPciNetDevice(device, np.rFactory, rc, selectorIndex); err == nil {
			newHostDevices = append(newHostDevices, newDevice)
		} else {
			glog.Errorf("netdevice GetDevices(): error creating new device: %q", err)
		}
	}
	return newHostDevices
}

func (np *netDeviceProvider) AddTargetDevices(devices []*ghw.PCIDevice, deviceCode int) error {
	for _, device := range devices {
		devClass, err := utils.ParseDeviceID(device.Class.ID)
		if err != nil {
			glog.Warningf("netdevice AddTargetDevices(): unable to parse device class for device %+v %q", device, err)
			continue
		}

		if devClass == int64(deviceCode) {
			vendorName := utils.NormalizeVendorName(device.Vendor.Name)
			productName := utils.NormalizeProductName(device.Product.Name)
			glog.Infof("netdevice AddTargetDevices(): device found: %-12s\t%-12s\t%-20s\t%-40s", device.Address,
				device.Class.ID, vendorName, productName)
			// exclude netdevice in-use in host
			if isDefaultRoute, _ := utils.HasDefaultRoute(device.Address); !isDefaultRoute {
				aPF := utils.IsSriovPF(device.Address)
				if aPF && utils.SriovConfigured(device.Address) {
					// do not add this device in net device list
					continue
				}
				np.deviceList = append(np.deviceList, device)
			}
		}
	}
	return nil
}

//nolint:gocyclo
func (np *netDeviceProvider) GetFilteredDevices(devices []types.HostDevice,
	rc *types.ResourceConfig, selectorIndex int) ([]types.HostDevice, error) {
	filteredDevice := devices
	if selectorIndex < 0 || selectorIndex >= len(rc.SelectorObjs) {
		return filteredDevice, fmt.Errorf("invalid selectorIndex %d, resource config only has %d selector objects",
			selectorIndex, len(rc.SelectorObjs))
	}
	nf, ok := rc.SelectorObjs[selectorIndex].(*types.NetDeviceSelectors)
	if !ok {
		return filteredDevice, fmt.Errorf("unable to convert SelectorObj to NetDeviceSelectors")
	}

	rf := np.rFactory

	// filter by vendor list
	filteredDevice = rf.FilterBySelector("vendors", nf.Vendors, filteredDevice)

	// filter by device list
	filteredDevice = rf.FilterBySelector("devices", nf.Devices, filteredDevice)

	// filter by driver list
	filteredDevice = rf.FilterBySelector("drivers", nf.Drivers, filteredDevice)

	// filter by pciAddresses list
	filteredDevice = rf.FilterBySelector("pciAddresses", nf.PciAddresses, filteredDevice)

	// filter by acpiIndexes list
	filteredDevice = rf.FilterBySelector("acpiIndexes", nf.AcpiIndexes, filteredDevice)

	// filter by PfNames list
	filteredDevice = rf.FilterBySelector("pfNames", nf.PfNames, filteredDevice)

	// filter by RootDevices list
	filteredDevice = rf.FilterBySelector("rootDevices", nf.RootDevices, filteredDevice)

	// filter by linkTypes list
	if len(nf.LinkTypes) > 1 {
		glog.Warningf("Link type selector should have a single value.")
	}
	filteredDevice = rf.FilterBySelector("linkTypes", nf.LinkTypes, filteredDevice)

	// filter by DDP Profiles list
	filteredDevice = rf.FilterBySelector("ddpProfiles", nf.DDPProfiles, filteredDevice)

	// filter by PKeys list
	filteredDevice = rf.FilterBySelector("pKeys", nf.PKeys, filteredDevice)

	// filter for rdma devices
	if nf.IsRdma {
		rdmaDevices := make([]types.HostDevice, 0)
		for _, dev := range filteredDevice {
			if dev.(types.NetDevice).IsRdma() {
				rdmaDevices = append(rdmaDevices, dev)
			}
		}
		filteredDevice = rdmaDevices
	}

	// filter for vDPA-capable devices
	if nf.VdpaType != "" {
		vdpaDevices := make([]types.HostDevice, 0)
		for _, dev := range filteredDevice {
			vdpaDev := dev.(types.PciNetDevice).GetVdpaDevice()
			if vdpaDev == nil {
				continue
			}
			if vType := vdpaDev.GetType(); vType != types.VdpaInvalidType && vType == nf.VdpaType {
				vdpaDevices = append(vdpaDevices, dev)
			}
		}
		filteredDevice = vdpaDevices
	}

	return filteredDevice, nil
}

// ValidConfig performs validation of NetDeviceSelectors
func (np *netDeviceProvider) ValidConfig(rc *types.ResourceConfig) bool {
	for _, selector := range rc.SelectorObjs {
		nf, ok := selector.(*types.NetDeviceSelectors)
		if !ok {
			glog.Errorf("unable to convert SelectorObj to NetDeviceSelectors")
			return false
		}
		if nf.IsRdma && nf.VdpaType != "" {
			glog.Errorf("invalid config: VdpaType and IsRdma are mutually exclusive options")
			return false
		}
	}
	return true
}
