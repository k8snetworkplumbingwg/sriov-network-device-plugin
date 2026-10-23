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
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

// ResourcePoolImpl implements stub ResourcePool interface
type ResourcePoolImpl struct {
	config       *types.ResourceConfig
	devicePool   map[string]types.HostDevice
	healthMu     sync.RWMutex
	nativeHealth map[string]string
}

var _ types.ResourcePool = &ResourcePoolImpl{}

// NewResourcePool returns an instance of resourcePool
func NewResourcePool(rc *types.ResourceConfig, devicePool map[string]types.HostDevice) *ResourcePoolImpl {
	rp := &ResourcePoolImpl{
		config:       rc,
		devicePool:   devicePool,
		nativeHealth: make(map[string]string),
	}
	rp.Probe()
	return rp
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

// GetDevices returns a map of Kubelet API devices
func (rp *ResourcePoolImpl) GetDevices() map[string]*pluginapi.Device {
	rp.healthMu.RLock()
	defer rp.healthMu.RUnlock()
	devices := make(map[string]*pluginapi.Device)
	for id, dev := range rp.devicePool {
		snapshot := *dev.GetAPIDevice()
		if health, ok := rp.nativeHealth[id]; ok {
			snapshot.Health = health
		}
		devices[id] = &snapshot
	}
	return devices
}

// Probe checks native VFIO access without opening devices or acquiring DMA ownership.
func (rp *ResourcePoolImpl) Probe() bool {
	if rp.config.VFIOMode != types.VFIOModeIOMMUFD {
		return false
	}
	rp.healthMu.Lock()
	defer rp.healthMu.Unlock()
	changed := false
	for id, dev := range rp.devicePool {
		health := pluginapi.Healthy
		_, err := dev.GetDeviceSpecs()
		if err != nil {
			health = pluginapi.Unhealthy
		}
		if rp.nativeHealth[id] != health {
			if err != nil {
				glog.Warningf("VFIO resource %s device %s is unhealthy: %v", rp.GetResourceName(), id, err)
			}
			changed = true
			rp.nativeHealth[id] = health
		}
	}
	return changed
}

// GetDeviceSpecs returns list of plugin API device specs for a list of device IDs
func (rp *ResourcePoolImpl) GetDeviceSpecs(deviceIDs []string) ([]*pluginapi.DeviceSpec, error) {
	glog.Infof("GetDeviceSpecs(): for devices: %v", deviceIDs)
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	// Resolve all devices before returning any allocation paths.
	for _, id := range deviceIDs {
		dev, ok := rp.devicePool[id]
		if !ok {
			return nil, fmt.Errorf("resource %s does not contain device %s", rp.GetResourceName(), id)
		}
		newSpecs, err := dev.GetDeviceSpecs()
		if err != nil {
			return nil, err
		}
		for _, ds := range newSpecs {
			if !rp.DeviceSpecExist(devSpecs, ds) {
				devSpecs = append(devSpecs, ds)
			}
		}
	}
	return devSpecs, nil
}

// GetEnvs returns a map with two keys.
// environment variable key base on PCIDEVICE_<prefix>_<resource-name> with a list of allocated pci addresses
// environment variable key base on PCIDEVICE_<prefix>_<resource-name>_INFO that contains info from all the
// requested info providers for every pci address allocated
func (rp *ResourcePoolImpl) GetEnvs(prefix string, deviceIDs []string) (map[string]string, error) {
	glog.Infof("GetEnvs(): for devices: %v", deviceIDs)
	devInfos := make(map[string]map[string]types.AdditionalInfo, 0)
	IDList := []string{}
	// Consolidates all ExtraEnvVariables
	for _, id := range deviceIDs {
		if dev, ok := rp.devicePool[id]; ok {
			envs := dev.GetEnvVal()
			devInfos[id] = envs
			IDList = append(IDList, id)
		}
	}

	envs := make(map[string]string)

	// construct PCIDEVICE_<prefix>_<resource-name> environment variable
	key := fmt.Sprintf("%s_%s_%s", "PCIDEVICE", prefix, rp.GetResourceName())
	key = strings.ToUpper(strings.Replace(key, ".", "_", -1))
	envs[key] = strings.Join(IDList, ",")

	// construct PCIDEVICE_<prefix>_<resource-name>_INFO environment variable
	key = fmt.Sprintf("%s_%s_%s_INFO", "PCIDEVICE", prefix, rp.GetResourceName())
	key = strings.ToUpper(strings.Replace(key, ".", "_", -1))
	envData, err := json.Marshal(devInfos)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal environment variable object: %v", err)
	}
	envs[key] = string(envData)

	return envs, nil
}

// GetMounts returns a list of Mount for device IDs
func (rp *ResourcePoolImpl) GetMounts(deviceIDs []string) []*pluginapi.Mount {
	glog.Infof("GetMounts(): for devices: %v", deviceIDs)
	devMounts := make([]*pluginapi.Mount, 0)

	for _, id := range deviceIDs {
		if dev, ok := rp.devicePool[id]; ok {
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

// GetDevicePool returns HostDevice pool as a map
func (rp *ResourcePoolImpl) GetDevicePool() map[string]types.HostDevice {
	return rp.devicePool
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
