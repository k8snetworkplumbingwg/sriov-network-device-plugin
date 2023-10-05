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

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

// GenPciDevice implementation
type GenPciDevice struct {
	pciAddr   string
	acpiIndex string
}

// NewGenPciDevice returns GenPciDevice instance
func NewGenPciDevice(dev *ghw.PCIDevice) (*GenPciDevice, error) {
	pciAddr := dev.Address

	acpiIndex, err := utils.GetAcpiIndex(dev.Address)
	if err != nil {
		return nil, err
	}

	return &GenPciDevice{
		pciAddr:   pciAddr,
		acpiIndex: acpiIndex,
	}, nil
}

// GetPciAddr returns pci address of the device
func (pd *GenPciDevice) GetPciAddr() string {
	return pd.pciAddr
}

// GetAcpiIndex returns ACPI index of the device
func (pd *GenPciDevice) GetAcpiIndex() string {
	return pd.acpiIndex
}
