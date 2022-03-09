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
	"github.com/jaypipes/ghw"
	"github.com/vishvananda/netlink"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

const (
	maxVendorNameLen  = 20
	maxProductNameLen = 40
	classIDBaseInt    = 16
	classIDBitSize    = 64
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

func (np *netDeviceProvider) GetDevices(rc *types.ResourceConfig) []types.PciDevice {
	newPciDevices := make([]types.PciDevice, 0)
	nf, ok := rc.SelectorObj.(*types.NetDeviceSelectors)
	if !ok {
		glog.Errorf("netdevice GetDevices(): unable to convert SelectorObj to NetDeviceSelectors")
		return newPciDevices
	}

	if len(nf.AuxDevices) == 0 {
		glog.Infof("netdevice GetDevices(): processing PciNetDevices")
		for _, device := range np.deviceList {
			if newDevice, err := NewPciNetDevice(device, np.rFactory, rc); err == nil {
				newPciDevices = append(newPciDevices, newDevice)
			} else {
				glog.Errorf("netdevice GetDevices(): error creating new device: %q", err)
			}
		}
	} else {
		glog.Infof("netdevice GetDevices(): processing AuxNetDevices")
		for _, device := range np.deviceList {
			if auxDevices, err := NewAuxNetDevices(device, np.rFactory, rc); err == nil {
				newPciDevices = append(newPciDevices, auxDevices...)
			} else {
				glog.Errorf("netdevice GetDevices(): failed to get auxiliary devices: %q", err)
			}
		}
	}
	return newPciDevices
}

func (np *netDeviceProvider) AddTargetDevices(devices []*ghw.PCIDevice, deviceCode int) error {
	for _, device := range devices {
		devClass, err := strconv.ParseInt(device.Class.ID, classIDBaseInt, classIDBitSize)
		if err != nil {
			glog.Warningf("netdevice AddTargetDevices(): unable to parse device class for device %+v %q", device, err)
			continue
		}

		if devClass == int64(deviceCode) {
			vendor := device.Vendor
			vendorName := vendor.Name
			if len(vendor.Name) > maxVendorNameLen {
				vendorName = string([]byte(vendorName)[0:17]) + "..."
			}
			product := device.Product
			productName := product.Name
			if len(product.Name) > maxProductNameLen {
				productName = string([]byte(productName)[0:37]) + "..."
			}
			glog.Infof("netdevice AddTargetDevices(): device found: %-12s\t%-12s\t%-20s\t%-40s", device.Address,
				device.Class.ID, vendorName, productName)
			// exclude netdevice in-use in host
			if isDefaultRoute, _ := hasDefaultRoute(device.Address); !isDefaultRoute {
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

// hasDefaultRoute returns true if PCI network device is default route interface
func hasDefaultRoute(pciAddr string) (bool, error) {
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
				continue
			}

			routes, err := netlink.RouteList(link, netlink.FAMILY_V4) // IPv6 routes: all interface has at least one link local route entry
			if err != nil {
				glog.Errorf("failed to get routes for interface: %s, %q", ifName, err)
				continue
			}
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

//nolint:gocyclo
func (np *netDeviceProvider) GetFilteredDevices(devices []types.PciDevice, rc *types.ResourceConfig) ([]types.PciDevice, error) {
	filteredDevice := devices
	nf, ok := rc.SelectorObj.(*types.NetDeviceSelectors)
	if !ok {
		return filteredDevice, fmt.Errorf("unable to convert SelectorObj to NetDeviceSelectors")
	}

	rf := np.rFactory
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

	// filter by pciAddresses list
	if nf.PciAddresses != nil && len(nf.PciAddresses) > 0 {
		if selector, err := rf.GetSelector("pciAddresses", nf.PciAddresses); err == nil {
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

	// filter by DDP Profiles list
	if nf.DDPProfiles != nil && len(nf.DDPProfiles) > 0 {
		if selector, err := rf.GetSelector("ddpProfiles", nf.DDPProfiles); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// filter for rdma devices
	if nf.IsRdma {
		rdmaDevices := make([]types.PciDevice, 0)
		for _, dev := range filteredDevice {
			if dev.(types.PciNetDevice).GetRdmaSpec().IsRdma() {
				rdmaDevices = append(rdmaDevices, dev)
			}
		}
		filteredDevice = rdmaDevices
	}

	// filter for vDPA-capable devices
	if nf.VdpaType != "" {
		vdpaDevices := make([]types.PciDevice, 0)
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

	if len(nf.AuxDevices) > 0 {
		if selector, err := rf.GetSelector("auxDevices", nf.AuxDevices); err == nil {
			filteredDevice = selector.Filter(filteredDevice)
		}
	}

	// convert []PciNetDevice to []PciDevice
	newDeviceList := make([]types.PciDevice, len(filteredDevice))
	copy(newDeviceList, filteredDevice)

	return newDeviceList, nil
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
