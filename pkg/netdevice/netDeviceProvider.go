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

func (np *netDeviceProvider) GetDevices(rc *types.ResourceConfig) []types.HostDevice {
	newHostDevices := make([]types.HostDevice, 0)
	for _, device := range np.deviceList {
		if newDevice, err := NewPciNetDevice(device, np.rFactory, rc); err == nil {
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
func (np *netDeviceProvider) GetFilteredDevices(devices []types.HostDevice, rc *types.ResourceConfig) ([]types.HostDevice, error) {
	filteredDevice := devices
	nf, ok := rc.SelectorObj.(*types.NetDeviceSelectors)
	if !ok {
		return filteredDevice, fmt.Errorf("unable to convert SelectorObj to NetDeviceSelectors")
	}

	selectors := np.GetSelectors(nf)
	for _, selector := range selectors {
		filteredDevice = selector.Filter(filteredDevice)
	}

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

//nolint:gocyclo
func (np *netDeviceProvider) GetSelectors(nf *types.NetDeviceSelectors) []types.DeviceSelector {
	selectors := []types.DeviceSelector{}
	rf := np.rFactory
	// filter by vendor list
	if nf.Vendors != nil && len(nf.Vendors) > 0 {
		if selector, err := rf.GetSelector("vendors", nf.Vendors); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by device list
	if nf.Devices != nil && len(nf.Devices) > 0 {
		if selector, err := rf.GetSelector("devices", nf.Devices); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by driver list
	if nf.Drivers != nil && len(nf.Drivers) > 0 {
		if selector, err := rf.GetSelector("drivers", nf.Drivers); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by pciAddresses list
	if nf.PciAddresses != nil && len(nf.PciAddresses) > 0 {
		if selector, err := rf.GetSelector("pciAddresses", nf.PciAddresses); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by PfNames list
	if nf.PfNames != nil && len(nf.PfNames) > 0 {
		if selector, err := rf.GetSelector("pfNames", nf.PfNames); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by NicNames list
	if nf.NicNames != nil && len(nf.NicNames) > 0 {
		if selector, err := rf.GetSelector("nicNames", nf.NicNames); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by RootDevices list
	if nf.RootDevices != nil && len(nf.RootDevices) > 0 {
		if selector, err := rf.GetSelector("rootDevices", nf.RootDevices); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by linkTypes list
	if nf.LinkTypes != nil && len(nf.LinkTypes) > 0 {
		if len(nf.LinkTypes) > 1 {
			glog.Warningf("Link type selector should have a single value.")
		}
		if selector, err := rf.GetSelector("linkTypes", nf.LinkTypes); err == nil {
			selectors = append(selectors, selector)
		}
	}

	// filter by DDP Profiles list
	if nf.DDPProfiles != nil && len(nf.DDPProfiles) > 0 {
		if selector, err := rf.GetSelector("ddpProfiles", nf.DDPProfiles); err == nil {
			selectors = append(selectors, selector)
		}
	}

	return selectors
}

// ValidConfig performs validation of NetDeviceSelectors
func (np *netDeviceProvider) ValidConfig(rc *types.ResourceConfig) bool {
	nf, ok := rc.SelectorObj.(*types.NetDeviceSelectors)
	if !ok {
		glog.Errorf("unable to convert SelectorObj to NetDeviceSelectors")
		return false
	}
	if nf.IsRdma && nf.VdpaType != "" {
		glog.Errorf("invalid config: VdpaType and IsRdma are mutually exclusive options")
		return false
	}
	return true
}
