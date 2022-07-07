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

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// ResourcePoolImpl implements stub ResourcePool interface
type ResourcePoolImpl struct {
	config         *types.ResourceConfig
	devicePool     []types.PciDevice
	deviceProvider types.DeviceProvider
	allocated      *map[string]bool
}

var _ types.ResourcePool = &ResourcePoolImpl{}

// NewResourcePool returns an instance of resourcePool
func NewResourcePool(rc *types.ResourceConfig, deviceProvider types.DeviceProvider, allocated *map[string]bool) *ResourcePoolImpl {
	return &ResourcePoolImpl{
		config:         rc,
		deviceProvider: deviceProvider,
		allocated:      allocated,
	}
}

// GetConfig returns ResourceConfig for this resourcePool
func (rp *ResourcePoolImpl) GetConfig() *types.ResourceConfig {
	return rp.config
}

// InitDevice - not implemented
func (rp *ResourcePoolImpl) InitDevice() error {
	// Not implemented
	return nil
}

// GetResourceName returns the resource name as string
func (rp *ResourcePoolImpl) GetResourceName() string {
	return rp.config.ResourceName
}

// GetResourcePrefix returns the resource name prefix as string
func (rp *ResourcePoolImpl) GetResourcePrefix() string {
	return rp.config.ResourcePrefix
}

func pciDeviceToPluginapiDevice(toConvert []types.PciDevice) map[string]*pluginapi.Device {
	apiDevices := make(map[string]*pluginapi.Device, 0)

	for _, dev := range toConvert {
		pciAddr := dev.GetPciAddr()
		apiDevices[pciAddr] = dev.GetAPIDevice()
	}
	return apiDevices
}

// GetDevices returns a map of Kubelet API devices
func (rp *ResourcePoolImpl) GetDevices() map[string]*pluginapi.Device {
	return pciDeviceToPluginapiDevice(rp.devicePool)
}

// Probe - does device healthcheck. Not implemented
func (rp *ResourcePoolImpl) Probe() bool {
	devToString := func(d types.PciDevice) string {
		return fmt.Sprintf("%s,%s,%s\n", d.GetPciAddr(), d.GetVendor(), d.GetDriver())
	}

	toSet := func(l []types.PciDevice) map[string]types.PciDevice {
		ret := make(map[string]types.PciDevice, 0)
		for _, d := range l {
			ret[devToString(d)] = d
		}
		return ret
	}

	changed := false
	rp.deviceProvider.DiscoverDevices()
	devices := rp.deviceProvider.GetDevices(rp.config)
	filteredDevices, _ := rp.deviceProvider.GetFilteredDevices(devices, rp.config)

	filteredDevicesMap := toSet(filteredDevices)
	devicePoolMap := toSet(rp.devicePool)

	for k, d := range filteredDevicesMap {
		if _, ok := devicePoolMap[k]; !ok {
			if _, ok := (*rp.allocated)[devToString(d)]; !ok {
				glog.Infof("ResourcePool probe(): new device found: %-12s\t%-12s", d.GetPciAddr(), d.GetVendor())
				(*rp.allocated)[devToString(d)] = true
				changed = true
			}
		}
	}

	for k, d := range devicePoolMap {
		if _, ok := filteredDevicesMap[k]; !ok {
			glog.Infof("ResourcePool probe(): old device removed: %-12s\t%-12s", d.GetPciAddr(), d.GetVendor())
			changed = true
			delete(*rp.allocated, devToString(d))
		}
	}

	if changed {
		rp.devicePool = filteredDevices
	}

	return changed
}

func (rp *ResourcePoolImpl) getDeviceWithID() map[string]types.PciDevice {
	deviceWithID := make(map[string]types.PciDevice, 0)

	for _, dev := range rp.devicePool {
		pciAddr := dev.GetPciAddr()
		deviceWithID[pciAddr] = dev
	}
	return deviceWithID
}

// GetDeviceSpecs returns list of plugin API device specs for a list of device IDs
func (rp *ResourcePoolImpl) GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec {
	glog.Infof("GetDeviceSpecs(): for devices: %v", deviceIDs)
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	deviceWithID := rp.getDeviceWithID()

	// Add vfio group specific devices
	for _, id := range deviceIDs {
		if dev, ok := deviceWithID[id]; ok {
			newSpecs := dev.GetDeviceSpecs()
			for _, ds := range newSpecs {
				if !rp.DeviceSpecExist(devSpecs, ds) {
					devSpecs = append(devSpecs, ds)
				}
			}
		}
	}
	return devSpecs
}

// GetEnvs returns a list of device specific Env values for device IDs
func (rp *ResourcePoolImpl) GetEnvs(deviceIDs []string) []string {
	glog.Infof("GetEnvs(): for devices: %v", deviceIDs)
	devEnvs := make([]string, 0)

	deviceWithID := rp.getDeviceWithID()

	// Consolidates all Envs
	for _, id := range deviceIDs {
		if dev, ok := deviceWithID[id]; ok {
			env := dev.GetEnvVal()
			devEnvs = append(devEnvs, env)
		}
	}

	return devEnvs
}

// GetMounts returns a list of Mount for device IDs
func (rp *ResourcePoolImpl) GetMounts(deviceIDs []string) []*pluginapi.Mount {
	glog.Infof("GetMounts(): for devices: %v", deviceIDs)
	devMounts := make([]*pluginapi.Mount, 0)

	deviceWithID := rp.getDeviceWithID()

	for _, id := range deviceIDs {
		if dev, ok := deviceWithID[id]; ok {
			mnt := dev.GetMounts()
			devMounts = append(devMounts, mnt...)
		}
	}
	return devMounts
}

// DeviceSpecExist checks if a DeviceSpec already exist in a DeviceSpec list
func (rp *ResourcePoolImpl) DeviceSpecExist(specs []*pluginapi.DeviceSpec, newSpec *pluginapi.DeviceSpec) bool {
	for _, sp := range specs {
		if sp.HostPath == newSpec.HostPath {
			return true
		}
	}
	return false
}

// GetDevicePool returns PciDevice pool as a map
func (rp *ResourcePoolImpl) GetDevicePool() map[string]types.PciDevice {
	deviceWithID := make(map[string]types.PciDevice, 0)

	for _, dev := range rp.devicePool {
		pciAddr := dev.GetPciAddr()
		deviceWithID[pciAddr] = dev
	}
	return deviceWithID
}

// StoreDeviceInfoFile does nothing. DeviceType-specific ResourcePools might
// store information according to the k8snetworkplumbingwg/device-info-spec
func (rp *ResourcePoolImpl) StoreDeviceInfoFile(resourceNamePrefix string) error {
	return nil
}

// CleanDeviceInfoFile does nothing. DeviceType-specific ResourcePools might
// clean the Device Info file
func (rp *ResourcePoolImpl) CleanDeviceInfoFile(resourceNamePrefix string) error {
	return nil
}
