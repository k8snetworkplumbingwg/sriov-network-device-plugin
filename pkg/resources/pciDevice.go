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

package resources

import (
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	"github.com/jaypipes/ghw"
)

type pciDevice struct {
	basePciDevice *ghw.PCIDevice
	pfAddr        string
	driver        string
	vendor        string
	product       string
	vfID          int
	numa          string
}

// NewPciDevice returns an instance of PciDevice interface
func NewPciDevice(dev *ghw.PCIDevice, rFactory types.ResourceFactory) (types.PciDevice, error) {

	// Get driver info
	pciAddr := dev.Address
	driverName, err := utils.GetDriverName(pciAddr)
	if err != nil {
		return nil, err
	}

	vfID, err := utils.GetVFID(pciAddr)
	if err != nil {
		return nil, err
	}

	nodeNum := utils.GetDevNode(pciAddr)
	// 	Create pciNetDevice object with all relevent info
	return &pciDevice{
		basePciDevice: dev,
		driver:        driverName,
		vfID:          vfID,
		numa:          nodeToStr(nodeNum),
	}, nil
}

func (pd *pciDevice) GetPfPciAddr() string {
	return pd.pfAddr
}

func (pd *pciDevice) GetVendor() string {
	return pd.basePciDevice.Vendor.ID
}

func (pd *pciDevice) GetDeviceCode() string {
	return pd.basePciDevice.Product.ID
}

func (pd *pciDevice) GetPciAddr() string {
	return pd.basePciDevice.Address
}

func (pd *pciDevice) GetDriver() string {
	return pd.driver
}

func (pd *pciDevice) IsSriovPF() bool {
	return false
}

func (pd *pciDevice) GetSubClass() string {
	return pd.basePciDevice.Subclass.ID
}

func (pd *pciDevice) GetVFID() int {
	return pd.vfID
}

func (pd *pciDevice) GetNumaInfo() string {
	return pd.numa
}
