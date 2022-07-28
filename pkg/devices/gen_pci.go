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
)

// GenPciDevice implementation
type GenPciDevice struct {
	pciAddr string
}

// NewGenPciDevice returns GenPciDevice instance
func NewGenPciDevice(dev *ghw.PCIDevice) (*GenPciDevice, error) {
	pciAddr := dev.Address

	return &GenPciDevice{
		pciAddr: pciAddr,
	}, nil
}

// GetPciAddr returns pci address of the device
func (pd *GenPciDevice) GetPciAddr() string {
	return pd.pciAddr
}
