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

package devices

import (
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

// APIDeviceImpl is an implementation of APIDevice interface
type APIDeviceImpl struct {
	device        *pluginapi.Device
	infoProviders []types.DeviceInfoProvider
}

// NewAPIDeviceImpl returns an instance implementation of APIDevice interface
func NewAPIDeviceImpl(id string, infoProviders []types.DeviceInfoProvider, nodeNum int) *APIDeviceImpl {
	var topology *pluginapi.TopologyInfo
	if nodeNum >= 0 {
		topology = &pluginapi.TopologyInfo{
			Nodes: []*pluginapi.NUMANode{
				{ID: int64(nodeNum)},
			},
		}
	}
	return &APIDeviceImpl{
		device: &pluginapi.Device{
			ID:       id,
			Health:   pluginapi.Healthy,
			Topology: topology,
		},
		infoProviders: infoProviders,
	}
}

// GetDeviceSpecs returns list of device specs
func (ad *APIDeviceImpl) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	dSpecs := make([]*pluginapi.DeviceSpec, 0)
	for _, infoProvider := range ad.infoProviders {
		dSpecs = append(dSpecs, infoProvider.GetDeviceSpecs()...)
	}

	return dSpecs
}

// GetEnvVal returns device environment variables
func (ad *APIDeviceImpl) GetEnvVal() map[string]types.AdditionalInfo {
	envValMap := make(map[string]types.AdditionalInfo, 0)
	for _, provider := range ad.infoProviders {
		envValMap[provider.GetName()] = provider.GetEnvVal()
	}
	return envValMap
}

// GetMounts returns list of device host mounts
func (ad *APIDeviceImpl) GetMounts() []*pluginapi.Mount {
	mnt := make([]*pluginapi.Mount, 0)
	for _, infoProvider := range ad.infoProviders {
		mnt = append(mnt, infoProvider.GetMounts()...)
	}

	return mnt
}

// GetAPIDevice returns k8s API device
func (ad *APIDeviceImpl) GetAPIDevice() *pluginapi.Device {
	return ad.device
}
