/*
 * SPDX-FileCopyrightText: Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package auxnetdevice

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type auxNetDeviceProvider struct {
	deviceList []*ghw.PCIDevice
	rFactory   types.ResourceFactory
}

// NewAuxNetDeviceProvider DeviceProvider implementation from auxNetDeviceProvider instance
func NewAuxNetDeviceProvider(rf types.ResourceFactory) types.DeviceProvider {
	return &auxNetDeviceProvider{
		rFactory:   rf,
		deviceList: make([]*ghw.PCIDevice, 0),
	}
}

func (ap *auxNetDeviceProvider) GetDiscoveredDevices() []*ghw.PCIDevice {
	return ap.deviceList
}

func (ap *auxNetDeviceProvider) GetDevices(rc *types.ResourceConfig) []types.HostDevice {
	newAuxDevices := make([]types.HostDevice, 0)
	for _, device := range ap.deviceList {
		// discover auxiliary device names
		auxDevs, err := utils.GetSriovnetProvider().GetAuxNetDevicesFromPci(device.Address)
		if err == nil {
			if len(auxDevs) == 0 {
				glog.Warningf("auxnetdevice GetDevices(): no auxiliary devices for PCI %s", device.Address)
				continue
			}
			for _, auxDev := range auxDevs {
				if newDevice, err := NewAuxNetDevice(device, auxDev, ap.rFactory, rc); err == nil {
					newAuxDevices = append(newAuxDevices, newDevice)
				} else {
					glog.Warningf("auxnetdevice GetDevices(): error creating new device %s PCI %s: %q",
						auxDev, device.Address, err)
				}
			}
		} else {
			glog.Warningf("auxNetDevice GetDevices(): error getting auxnetdevices from device %s: %q",
				device.Address, err)
		}
	}
	return newAuxDevices
}

func (ap *auxNetDeviceProvider) AddTargetDevices(devices []*ghw.PCIDevice, deviceCode int) error {
	for _, device := range devices {
		devClass, err := utils.ParseDeviceID(device.Class.ID)
		if err != nil {
			glog.Warningf("auxNetDevice AddTargetDevices(): unable to parse device class for device %+v %q", device, err)
			continue
		}

		if devClass == int64(deviceCode) {
			vendorName := utils.NormalizeVendorName(device.Vendor.Name)
			productName := utils.NormalizeProductName(device.Product.Name)
			glog.Infof("auxnetdevice AddTargetDevices(): device found: %-12s\t%-12s\t%-20s\t%-40s", device.Address,
				device.Class.ID, vendorName, productName)
			if isDefaultRoute, _ := utils.HasDefaultRoute(device.Address); !isDefaultRoute {
				ap.deviceList = append(ap.deviceList, device)
			}
		}
	}
	return nil
}

//nolint:gocyclo
func (ap *auxNetDeviceProvider) GetFilteredDevices(devices []types.HostDevice, rc *types.ResourceConfig) ([]types.HostDevice, error) {
	filteredDevice := devices
	nf, ok := rc.SelectorObj.(*types.AuxNetDeviceSelectors)
	if !ok {
		return filteredDevice, fmt.Errorf("unable to convert SelectorObj to AuxNetDeviceSelectors")
	}

	rf := ap.rFactory
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

	// filter by auxiliary device type list
	if nf.AuxTypes != nil && len(nf.AuxTypes) > 0 {
		if selector, err := rf.GetSelector("auxTypes", nf.AuxTypes); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by PfNames list
	if nf.PfNames != nil && len(nf.PfNames) > 0 {
		if selector, err := rf.GetSelector("pfNames", nf.PfNames); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter by RootDevices list
	if nf.RootDevices != nil && len(nf.RootDevices) > 0 {
		if selector, err := rf.GetSelector("rootDevices", nf.RootDevices); err == nil {
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

	return filteredDevice, nil
}

// ValidConfig performs validation of AuxNetDeviceSelectors
func (ap *auxNetDeviceProvider) ValidConfig(rc *types.ResourceConfig) bool {
	nf, ok := rc.SelectorObj.(*types.AuxNetDeviceSelectors)
	if !ok {
		glog.Errorf("unable to convert SelectorObj to AuxNetDeviceSelectors")
		return false
	}
	if len(nf.AuxTypes) == 0 {
		glog.Errorf("AuxTypes are not specified")
		return false
	}
	// Check that only supported auxiliary device types are specified
	// TODO ATM only SFs are supported; review this in the future if new types are added
	for _, auxType := range nf.AuxTypes {
		if auxType != "sf" {
			glog.Errorf("Only \"sf\" auxiliary device type currently supported")
			return false
		}
	}
	return true
}
