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
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

// HostDeviceImpl is an implementation of HostDevice interface
type HostDeviceImpl struct {
	types.APIDevice
	vendorID   string
	deviceCode string
	driver     string
}

// NewHostDeviceImpl returns an instance implementation of HostDevice interface
// A list of DeviceInfoProviders can be set externally.
// If empty, the default driver-based selection provided by ResourceFactory will be used
func NewHostDeviceImpl(dev *ghw.PCIDevice, deviceID string, rFactory types.ResourceFactory,
	rc *types.ResourceConfig, infoProviders []types.DeviceInfoProvider) (*HostDeviceImpl, error) {
	// Get driver info
	driverName, err := utils.GetDriverName(dev.Address)
	if err != nil {
		return nil, err
	}

	// Use the default Information Provided if not
	if len(infoProviders) == 0 {
		infoProviders = rFactory.GetDefaultInfoProvider(deviceID, driverName)
		if rc.AdditionalInfo != nil {
			infoProviders = append(infoProviders, infoprovider.NewExtraInfoProvider(dev.Address, rc.AdditionalInfo))
		}
	}

	nodeNum := -1
	if !rc.ExcludeTopology {
		nodeNum = utils.GetDevNode(dev.Address)
	}

	apiDevice := NewAPIDeviceImpl(deviceID, infoProviders, nodeNum)

	return &HostDeviceImpl{
		APIDevice:  apiDevice,
		vendorID:   dev.Vendor.ID,
		deviceCode: dev.Product.ID,
		driver:     driverName,
	}, nil
}

// GetVendor returns vendor identifier number of the device
func (hd *HostDeviceImpl) GetVendor() string {
	return hd.vendorID
}

// GetDeviceCode returns identifier number of the device
func (hd *HostDeviceImpl) GetDeviceCode() string {
	return hd.deviceCode
}

// GetDeviceID returns device unique identifier
func (hd *HostDeviceImpl) GetDeviceID() string {
	return hd.GetAPIDevice().ID
}

// GetDriver returns driver name of the device
func (hd *HostDeviceImpl) GetDriver() string {
	return hd.driver
}
