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
	"strconv"

	"github.com/jaypipes/ghw"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

type pciDevice struct {
	basePciDevice *ghw.PCIDevice
	pfAddr        string
	driver        string
	vendor        string
	product       string
	vfID          int
	env           string
	numa          string
	apiDevice     *pluginapi.Device
	deviceSpecs   []*pluginapi.DeviceSpec
	mounts        []*pluginapi.Mount
}

// Convert NUMA node number to string.
// A node of -1 represents "unknown" and is converted to the empty string.
func nodeToStr(nodeNum int) string {
	if nodeNum >= 0 {
		return strconv.Itoa(nodeNum)
	}
	return ""
}

// NewPciDevice returns an instance of PciDevice interface
func NewPciDevice(dev *ghw.PCIDevice, rFactory types.ResourceFactory) (types.PciDevice, error) {

	pciAddr := dev.Address

	// Get PF PCI address
	pfAddr, err := utils.GetPfAddr(pciAddr)
	if err != nil {
		return nil, err
	}

	// Get driver info
	driverName, err := utils.GetDriverName(pciAddr)
	if err != nil {
		return nil, err
	}

	vfID, err := utils.GetVFID(pciAddr)
	if err != nil {
		return nil, err
	}

	// Get Device file info (e.g., uio, vfio specific)
	// Get DeviceInfoProvider using device driver
	infoProvider := rFactory.GetInfoProvider(driverName)
	dSpecs := infoProvider.GetDeviceSpecs(pciAddr)
	mnt := infoProvider.GetMounts(pciAddr)
	env := infoProvider.GetEnvVal(pciAddr)
	nodeNum := utils.GetDevNode(pciAddr)
	apiDevice := &pluginapi.Device{
		ID:     pciAddr,
		Health: pluginapi.Healthy,
	}
	if nodeNum >= 0 {
		numaInfo := &pluginapi.NUMANode{
			ID: int64(nodeNum),
		}
		apiDevice.Topology = &pluginapi.TopologyInfo{
			Nodes: []*pluginapi.NUMANode{numaInfo},
		}
	}

	// 	Create pciNetDevice object with all relevent info
	return &pciDevice{
		basePciDevice: dev,
		pfAddr:        pfAddr,
		driver:        driverName,
		vfID:          vfID,
		apiDevice:     apiDevice,
		deviceSpecs:   dSpecs,
		mounts:        mnt,
		env:           env,
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

func (pd *pciDevice) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	return pd.deviceSpecs
}

func (pd *pciDevice) GetEnvVal() string {
	return pd.env
}

func (pd *pciDevice) GetMounts() []*pluginapi.Mount {
	return pd.mounts
}

func (pd *pciDevice) GetAPIDevice() *pluginapi.Device {
	return pd.apiDevice
}

func (pd *pciDevice) GetVFID() int {
	return pd.vfID
}

func (pd *pciDevice) GetNumaInfo() string {
	return pd.numa
}
