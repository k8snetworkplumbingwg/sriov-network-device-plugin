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

package factory

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/accelerator"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/auxnetdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

type resourceFactory struct {
	endPointPrefix string
	endPointSuffix string
	pluginWatch    bool
}

var instance *resourceFactory

// NewResourceFactory returns an instance of Resource Server factory
func NewResourceFactory(prefix, suffix string, pluginWatch bool) types.ResourceFactory {
	if instance == nil {
		return &resourceFactory{
			endPointPrefix: prefix,
			endPointSuffix: suffix,
			pluginWatch:    pluginWatch,
		}
	}
	return instance
}

// GetResourceServer returns an instance of ResourceServer for a ResourcePool
func (rf *resourceFactory) GetResourceServer(rp types.ResourcePool) (types.ResourceServer, error) {
	if rp != nil {
		prefix := rf.endPointPrefix
		if prefixOverride := rp.GetResourcePrefix(); prefixOverride != "" {
			prefix = prefixOverride
		}
		return resources.NewResourceServer(prefix, rf.endPointSuffix, rf.pluginWatch, rp), nil
	}
	return nil, fmt.Errorf("factory: unable to get resource pool object")
}

// GetDefaultInfoProvider returns an instance of DeviceInfoProvider using name as string
func (rf *resourceFactory) GetDefaultInfoProvider(pciAddr, name string) []types.DeviceInfoProvider {
	deviceInfoProvidersList := []types.DeviceInfoProvider{infoprovider.NewGenericInfoProvider(pciAddr)}

	switch name {
	case "vfio-pci":
		deviceInfoProvidersList = append(deviceInfoProvidersList, infoprovider.NewVfioInfoProvider(pciAddr))
	case "uio", "igb_uio":
		deviceInfoProvidersList = append(deviceInfoProvidersList, infoprovider.NewUioInfoProvider(pciAddr))
	}
	return deviceInfoProvidersList
}

// GetSelector returns an instance of DeviceSelector using selector attribute string and its associated values
func (rf *resourceFactory) GetSelector(attr string, values []string) (types.DeviceSelector, error) {
	switch attr {
	case "vendors":
		return resources.NewVendorSelector(values), nil
	case "devices":
		return resources.NewDeviceSelector(values), nil
	case "drivers":
		return resources.NewDriverSelector(values), nil
	case "pciAddresses":
		return resources.NewPciAddressSelector(values), nil
	case "pfNames":
		return resources.NewPfNameSelector(values), nil
	case "rootDevices":
		return resources.NewRootDeviceSelector(values), nil
	case "linkTypes":
		return resources.NewLinkTypeSelector(values), nil
	case "ddpProfiles":
		return resources.NewDdpSelector(values), nil
	case "auxTypes":
		return resources.NewAuxTypeSelector(values), nil
	default:
		return nil, fmt.Errorf("GetSelector(): invalid attribute %s", attr)
	}
}

// GetResourcePool returns an instance of resourcePool
func (rf *resourceFactory) GetResourcePool(rc *types.ResourceConfig, filteredDevice []types.HostDevice) (types.ResourcePool, error) {
	devicePool := make(map[string]types.HostDevice)
	for _, dev := range filteredDevice {
		id := dev.GetDeviceID()
		devicePool[id] = dev
		glog.Infof("device added: [identifier: %s, vendor: %s, device: %s, driver: %s]",
			id,
			dev.GetVendor(),
			dev.GetDeviceCode(),
			dev.GetDriver())
	}

	var rPool types.ResourcePool
	var err error
	switch rc.DeviceType {
	case types.NetDeviceType:
		if len(filteredDevice) > 0 {
			if _, ok := filteredDevice[0].(types.PciNetDevice); ok {
				nadUtils := rf.GetNadUtils()
				rPool = netdevice.NewNetResourcePool(nadUtils, rc, devicePool)
			} else {
				err = fmt.Errorf("invalid device list for NetDeviceType")
			}
		}
	case types.AcceleratorType:
		if len(filteredDevice) > 0 {
			if _, ok := filteredDevice[0].(types.AccelDevice); ok {
				rPool = accelerator.NewAccelResourcePool(rc, devicePool)
			} else {
				err = fmt.Errorf("invalid device list for AcceleratorType")
			}
		}
	case types.AuxNetDeviceType:
		if len(filteredDevice) > 0 {
			if _, ok := filteredDevice[0].(types.AuxNetDevice); ok {
				rPool = auxnetdevice.NewAuxNetResourcePool(rc, devicePool)
			} else {
				err = fmt.Errorf("invalid device list for AuxNetDeviceType")
			}
		}
	default:
		err = fmt.Errorf("cannot create resourcePool: invalid device type %s", rc.DeviceType)
	}
	return rPool, err
}

func (rf *resourceFactory) GetRdmaSpec(dt types.DeviceType, deviceID string) types.RdmaSpec {
	//nolint: exhaustive
	switch dt {
	case types.NetDeviceType:
		return devices.NewRdmaSpec(deviceID)
	case types.AuxNetDeviceType:
		return devices.NewAuxRdmaSpec(deviceID)
	default:
		return nil
	}
}

func (rf *resourceFactory) GetVdpaDevice(pciAddr string) types.VdpaDevice {
	return devices.GetVdpaDevice(pciAddr)
}

// GetDeviceProvider returns an instance of DeviceProvider based on DeviceType
func (rf *resourceFactory) GetDeviceProvider(dt types.DeviceType) types.DeviceProvider {
	switch dt {
	case types.NetDeviceType:
		return netdevice.NewNetDeviceProvider(rf)
	case types.AcceleratorType:
		return accelerator.NewAccelDeviceProvider(rf)
	case types.AuxNetDeviceType:
		return auxnetdevice.NewAuxNetDeviceProvider(rf)
	default:
		return nil
	}
}

// GetDeviceFilter unmarshal the "selector" values from ResourceConfig and returns a slice of *DeviceSelectors based on
// DeviceType in the ResourceConfig
func (rf *resourceFactory) GetDeviceFilter(rc *types.ResourceConfig) ([]interface{}, error) {
	switch rc.DeviceType {
	case types.NetDeviceType:
		netDeviceArray := make([]types.NetDeviceSelectors, 1)

		if err := json.Unmarshal(*rc.Selectors, &netDeviceArray[0]); err != nil {
			if err = json.Unmarshal(*rc.Selectors, &netDeviceArray); err != nil {
				return nil, fmt.Errorf("error unmarshalling NetDevice selector bytes %v", err)
			}
			if len(netDeviceArray) == 0 {
				return nil, fmt.Errorf("error, need at least one NetDevice selector, got 0")
			}
		}

		glog.Infof("Net device selector for resource %s is %+v", rc.ResourceName, netDeviceArray)
		interfaceArray := make([]interface{}, len(netDeviceArray))
		for i := range netDeviceArray {
			interfaceArray[i] = &netDeviceArray[i]
		}
		return interfaceArray, nil
	case types.AcceleratorType:
		accelDeviceArray := make([]types.AccelDeviceSelectors, 1)

		if err := json.Unmarshal(*rc.Selectors, &accelDeviceArray[0]); err != nil {
			if err = json.Unmarshal(*rc.Selectors, &accelDeviceArray); err != nil {
				return nil, fmt.Errorf("error unmarshalling Accelerator selector bytes %v", err)
			}
			if len(accelDeviceArray) == 0 {
				return nil, fmt.Errorf("error, need at least one Accelerator selector, get 0")
			}
		}

		glog.Infof("Accelerator device selector for resource %s is %+v", rc.ResourceName, accelDeviceArray)
		interfaceArray := make([]interface{}, len(accelDeviceArray))
		for i := range accelDeviceArray {
			interfaceArray[i] = &accelDeviceArray[i]
		}
		return interfaceArray, nil
	case types.AuxNetDeviceType:
		auxNetDeviceArray := make([]types.AuxNetDeviceSelectors, 1)

		if err := json.Unmarshal(*rc.Selectors, &auxNetDeviceArray[0]); err != nil {
			if err = json.Unmarshal(*rc.Selectors, &auxNetDeviceArray); err != nil {
				return nil, fmt.Errorf("error unmarshalling aux net device selector bytes %v", err)
			}
			if len(auxNetDeviceArray) == 0 {
				return nil, fmt.Errorf("error, need at least one aux net device selector, get 0")
			}
		}

		glog.Infof("aux net device selector for resource %s is %+v", rc.ResourceName, auxNetDeviceArray)
		interfaceArray := make([]interface{}, len(auxNetDeviceArray))
		for i := range auxNetDeviceArray {
			interfaceArray[i] = &auxNetDeviceArray[i]
		}
		return interfaceArray, nil
	default:
		return nil, fmt.Errorf("unable to get deviceFilter, invalid deviceType %s", rc.DeviceType)
	}
}

// GetNadUtils returns an instance of NadUtils
func (rf *resourceFactory) GetNadUtils() types.NadUtils {
	return netdevice.NewNadUtils()
}
