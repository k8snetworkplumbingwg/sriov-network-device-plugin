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

package utils

import (
	"github.com/Mellanox/rdmamap"
)

// RdmaProvider is a wrapper type over rdmamap library
type RdmaProvider interface {
	GetRdmaDevicesForPcidev(pciAddr string) []string
	GetRdmaDevicesForAuxdev(deviceID string) []string
	GetRdmaCharDevices(rdmaDeviceName string) []string
}

type defaultRdmaProvider struct {
}

var rdmaProvider RdmaProvider = &defaultRdmaProvider{}

// SetRdmaProviderInst method would be used by unit tests in other packages
func SetRdmaProviderInst(inst RdmaProvider) {
	rdmaProvider = inst
}

// GetRdmaProvider will be invoked by functions in other packages that would need access to the vdpa library methods.
func GetRdmaProvider() RdmaProvider {
	return rdmaProvider
}

func (defaultRdmaProvider) GetRdmaDevicesForPcidev(pciAddr string) []string {
	return rdmamap.GetRdmaDevicesForPcidev(pciAddr)
}

func (defaultRdmaProvider) GetRdmaDevicesForAuxdev(deviceID string) []string {
	return rdmamap.GetRdmaDevicesForAuxdev(deviceID)
}

func (defaultRdmaProvider) GetRdmaCharDevices(rdmaDeviceName string) []string {
	return rdmamap.GetRdmaCharDevices(rdmaDeviceName)
}
