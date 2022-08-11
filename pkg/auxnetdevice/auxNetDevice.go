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

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

// auxNetDevice extends HostDevice and embedds GenNetDevice
type auxNetDevice struct {
	types.HostDevice
	devices.GenNetDevice
	auxType string
}

// NewAuxNetDevice returns an instance of AciNetDevice interface
func NewAuxNetDevice(dev *ghw.PCIDevice, deviceID string, rFactory types.ResourceFactory,
	rc *types.ResourceConfig) (types.AuxNetDevice, error) {
	driverName, err := utils.GetDriverName(dev.Address)
	if err != nil {
		return nil, err
	}

	infoProviders := make([]types.DeviceInfoProvider, 0)
	infoProviders = append(infoProviders, rFactory.GetDefaultInfoProvider(deviceID, driverName))
	isRdma := false
	nf, ok := rc.SelectorObj.(*types.AuxNetDeviceSelectors)
	if ok {
		if nf.IsRdma {
			rdmaSpec := rFactory.GetRdmaSpec(types.AuxNetDeviceType, deviceID)
			if rdmaSpec.IsRdma() {
				isRdma = true
				infoProviders = append(infoProviders, infoprovider.NewRdmaInfoProvider(rdmaSpec))
			} else {
				glog.Warningf("RDMA resources for %s not found. Are RDMA modules loaded?", deviceID)
			}
		}
	}

	hostDev, err := devices.NewHostDeviceImpl(dev, deviceID, rFactory, rc, infoProviders)
	if err != nil {
		return nil, err
	}

	netDev, err := devices.NewGenNetDevice(deviceID, types.AuxNetDeviceType, isRdma)
	if err != nil {
		return nil, err
	}

	auxType := utils.ParseAuxDeviceType(deviceID)
	if auxType == "" {
		return nil, fmt.Errorf("device ID %s doesn't represent auxuliary device", deviceID)
	}

	return &auxNetDevice{
		HostDevice:   hostDev,
		GenNetDevice: *netDev,
		auxType:      auxType,
	}, nil
}

func (ad *auxNetDevice) GetAuxType() string {
	return ad.auxType
}
