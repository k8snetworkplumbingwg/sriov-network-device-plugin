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

package infoprovider

import (
	"fmt"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

// vdpaInfoProvider is a DeviceInfoProvider that handles the API information of vdpa-capable devices.
type vdpaInfoProvider struct {
	dev      types.VdpaDevice
	vdpaType types.VdpaType
	vdpaPath string
}

// NewVdpaInfoProvider returns a new InfoProvider associated with the given VDPAInfo
func NewVdpaInfoProvider(vdpaType types.VdpaType, vdpaDev types.VdpaDevice) types.DeviceInfoProvider {
	vdpaInfoProvider := &vdpaInfoProvider{
		dev:      vdpaDev,
		vdpaType: vdpaType,
	}
	return vdpaInfoProvider
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (vip *vdpaInfoProvider) GetName() string {
	return "vdpa"
}

// GetDeviceSpecs returns the DeviceSpec slice
func (vip *vdpaInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if healthy, err := vip.isHealthy(); !healthy {
		glog.Errorf("GetDeviceSpecs(): vDPA is required in the configuration but device does not have a healthy vdpa device: %s",
			err)
		return nil
	}
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	// DeviceSpecs only required for vhost vdpa type as the
	if vip.vdpaType == types.VdpaVhostType {
		vdpaPath, err := vip.dev.GetPath()
		if err != nil {
			glog.Errorf("Unexpected error when fetching the vdpa device path: %s", err)
			return nil
		}
		devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
			HostPath:      vdpaPath,
			ContainerPath: vdpaPath,
			Permissions:   "rw",
		})
		vip.vdpaPath = vdpaPath
	}
	return devSpecs
}

// GetEnvVal returns the environment variable value
func (vip *vdpaInfoProvider) GetEnvVal() types.AdditionalInfo {
	envs := make(map[string]string, 0)
	if vip.vdpaPath != "" {
		envs["mount"] = vip.vdpaPath
	}

	return envs
}

// GetMounts returns the mount points (none for this InfoProvider)
func (vip *vdpaInfoProvider) GetMounts() []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// isHealthy returns whether the device's vDPA information is healthy
func (vip *vdpaInfoProvider) isHealthy() (bool, error) {
	if vip.dev == nil {
		return false, fmt.Errorf("no vDPA device found")
	}

	if _, ok := types.SupportedVdpaTypes[vip.vdpaType]; !ok {
		return false, fmt.Errorf("vdpaType not supported %s", vip.vdpaType)
	}
	vType := vip.dev.GetType()
	if vType == types.VdpaInvalidType {
		return false, fmt.Errorf("device does not have a valid vdpa types")
	}
	if vType != vip.vdpaType {
		return false, fmt.Errorf("wrong vdpa type. Config expects %s but device is %s",
			vip.vdpaType, vType)
	}
	return true, nil
}

// *****************************************************************
