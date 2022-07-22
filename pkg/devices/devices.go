// Copyright 2020 Intel Corp. All Rights Reserved.
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

package devices

import (
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type pciDevice struct {
	types.HostDevice
	pfAddr string
	vfID   int
}

// NewPciDevice returns an instance of PciDevice interface
func NewPciDevice(dev *ghw.PCIDevice, rFactory types.ResourceFactory, rc *types.ResourceConfig,
	infoProviders []types.DeviceInfoProvider) (types.PciDevice, error) {
	pciAddr := dev.Address

	// Get PF PCI address
	pfAddr, err := utils.GetPfAddr(pciAddr)
	if err != nil {
		return nil, err
	}

	hostDevice, err := NewHostDeviceImpl(dev, pciAddr, rFactory, rc, infoProviders)
	if err != nil {
		return nil, err
	}

	vfID, err := utils.GetVFID(pciAddr)
	if err != nil {
		return nil, err
	}

	// Create pciDevice object with all relevant info
	return &pciDevice{
		HostDevice: hostDevice,
		pfAddr:     pfAddr,
		vfID:       vfID,
	}, nil
}

func (pd *pciDevice) GetPfPciAddr() string {
	return pd.pfAddr
}

func (pd *pciDevice) GetPciAddr() string {
	return pd.GetDeviceID()
}

func (pd *pciDevice) GetVFID() int {
	return pd.vfID
}
