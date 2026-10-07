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

package accelerator

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

const (
	accelPoolType         = "net-accel"
	deviceHealthVerbosity = 4
)

type accelResourcePool struct {
	*resources.ResourcePoolImpl
}

var _ types.ResourcePool = &accelResourcePool{}

// NewAccelResourcePool returns an instance of resourcePool
func NewAccelResourcePool(rc *types.ResourceConfig, devicePool map[string]types.HostDevice) types.ResourcePool {
	rp := resources.NewResourcePool(rc, devicePool)
	return &accelResourcePool{
		ResourcePoolImpl: rp,
	}
}

// Overrides GetDeviceSpecs
func (rp *accelResourcePool) GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec {
	glog.Infof("GetDeviceSpecs(): for devices: %v", deviceIDs)
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	devicePool := rp.GetDevicePool()

	// Add device driver specific devices
	for _, id := range deviceIDs {
		if dev, ok := devicePool[id]; ok {
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

// GetCDIKind returns device kind for CDI spec
func (rp *accelResourcePool) GetCDIName() string {
	return accelPoolType
}

// UpdateDeviceProbeStatus updates device health based on availability checks
func (rp *accelResourcePool) UpdateDeviceProbeStatus() error {
	devicePool := rp.GetDevicePool()
	for deviceID, dev := range devicePool {
		if dev == nil {
			continue
		}
		apiDev := dev.GetAPIDevice()
		if apiDev == nil {
			continue
		}
		if !rp.isDeviceHealthy(deviceID) {
			apiDev.Health = pluginapi.Unhealthy
		} else {
			apiDev.Health = pluginapi.Healthy
		}
	}
	return nil
}

// isDeviceHealthy checks if an accelerator device is healthy
func (rp *accelResourcePool) isDeviceHealthy(deviceID string) bool {
	// Check if device file exists in /sys/bus/pci/devices
	sysPath := fmt.Sprintf("/sys/bus/pci/devices/%s", deviceID)
	if _, err := os.Stat(sysPath); err != nil {
		glog.V(deviceHealthVerbosity).Infof("Device %s not found in sysfs: %v", deviceID, err)
		return false
	}
	// Device exists and is accessible
	return true
}
