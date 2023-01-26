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
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type rdmaSpec struct {
	isSupportRdma bool
	deviceSpec    []*pluginapi.DeviceSpec
}

func newRdmaSpec(rdmaResources []string) types.RdmaSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	isSupportRdma := false
	if len(rdmaResources) > 0 {
		isSupportRdma = true
		for _, res := range rdmaResources {
			resRdmaDevices := utils.GetRdmaProvider().GetRdmaCharDevices(res)
			for _, rdmaDevice := range resRdmaDevices {
				deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
					HostPath:      rdmaDevice,
					ContainerPath: rdmaDevice,
					Permissions:   "rwm",
				})
			}
		}
	}

	return &rdmaSpec{isSupportRdma: isSupportRdma, deviceSpec: deviceSpec}
}

// NewRdmaSpec returns the RdmaSpec for PCI address
func NewRdmaSpec(pciAddr string) types.RdmaSpec {
	rdmaResources := utils.GetRdmaProvider().GetRdmaDevicesForPcidev(pciAddr)
	return newRdmaSpec(rdmaResources)
}

// NewAuxRdmaSpec returns the RdmaSpec for auxiliary device ID
func NewAuxRdmaSpec(deviceID string) types.RdmaSpec {
	rdmaResources := utils.GetRdmaProvider().GetRdmaDevicesForAuxdev(deviceID)
	return newRdmaSpec(rdmaResources)
}

func (r *rdmaSpec) IsRdma() bool {
	return r.isSupportRdma
}

func (r *rdmaSpec) GetRdmaDeviceSpec() []*pluginapi.DeviceSpec {
	return r.deviceSpec
}
