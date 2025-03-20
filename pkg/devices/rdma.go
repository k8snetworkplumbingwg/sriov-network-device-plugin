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
	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type rdmaSpec struct {
	deviceID   string
	deviceType types.DeviceType
}

// NewRdmaSpec returns the RdmaSpec
func NewRdmaSpec(dt types.DeviceType, id string) types.RdmaSpec {
	if dt == types.AcceleratorType {
		return nil
	}
	return &rdmaSpec{deviceID: id, deviceType: dt}
}

func (r *rdmaSpec) IsRdma() bool {
	if len(r.getRdmaResources()) > 0 {
		return true
	}
	var bus string
	//nolint: exhaustive
	switch r.deviceType {
	case types.NetDeviceType:
		bus = "pci"
	case types.AuxNetDeviceType:
		bus = "auxiliary"
	default:
		return false
	}
	// In case of exclusive RDMA, if the resource is assigned to a pod
	// the files used to check if the device support RDMA are removed from the host.
	// In order to still report the resource in this state,
	// netlink param "enable_rdma" is checked to verify if the device supports RDMA.
	// This scenario cann happen if the device is discovered, assigned to a pod and then the plugin is restarted.
	rdma, err := utils.HasRdmaParam(bus, r.deviceID)
	if err != nil {
		glog.Infof("HasRdmaParam(): unable to get Netlink RDMA param for device %s : %q", r.deviceID, err)
		return false
	}
	return rdma
}

func (r *rdmaSpec) getRdmaResources() []string {
	//nolint: exhaustive
	switch r.deviceType {
	case types.NetDeviceType:
		return utils.GetRdmaProvider().GetRdmaDevicesForPcidev(r.deviceID)
	case types.AuxNetDeviceType:
		return utils.GetRdmaProvider().GetRdmaDevicesForAuxdev(r.deviceID)
	default:
		return make([]string, 0)
	}
}

func (r *rdmaSpec) GetRdmaDeviceSpec() []*pluginapi.DeviceSpec {
	rdmaResources := r.getRdmaResources()
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	if len(rdmaResources) > 0 {
		for _, res := range rdmaResources {
			resRdmaDevices := utils.GetRdmaProvider().GetRdmaCharDevices(res)
			for _, rdmaDevice := range resRdmaDevices {
				deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
					HostPath:      rdmaDevice,
					ContainerPath: rdmaDevice,
					Permissions:   "rw",
				})
			}
		}
	}
	return deviceSpec
}

// GetRdmaDeviceName returns the rdma device name
func (r *rdmaSpec) GetRdmaDeviceName() string {
	rdmaResource := r.getRdmaResources()
	if len(rdmaResource) > 0 {
		return rdmaResource[0]
	}
	return ""
}
