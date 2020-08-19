// Copyright 2018 Intel Corp. All Rights Reserved.
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

package netdevice

import (
	"github.com/golang/glog"
	"github.com/jaypipes/ghw"

	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
)

// pciNetDevice extends pciDevice
type pciNetDevice struct {
	types.PciDevice
	ifName    string
	pfName    string
	pfAddr    string
	linkSpeed string
	rdmaSpec  types.RdmaSpec
	linkType  string
}

// NewPciNetDevice returns an instance of PciNetDevice interface
func NewPciNetDevice(dev *ghw.PCIDevice, rFactory types.ResourceFactory, rc *types.ResourceConfig) (types.PciNetDevice, error) {

	var ifName string
	pciDev, err := resources.NewPciDevice(dev, rFactory)
	if err != nil {
		return nil, err
	}

	pciAddr := pciDev.GetPciAddr()
	netDevs, _ := utils.GetNetNames(pciAddr)
	if len(netDevs) == 0 {
		ifName = ""
	} else {
		ifName = netDevs[0]
	}
	pfName, err := utils.GetPfName(pciAddr)
	if err != nil {
		glog.Warningf("unable to get PF name %q", err.Error())
	}
	pfAddr, err := utils.GetPfAddr(pciAddr)
	if err != nil {
		glog.Warningf("unable to get PF address %q", err.Error())
	}

	rdmaSpec := rFactory.GetRdmaSpec(dev.Address)
	nf, ok := rc.SelectorObj.(*types.NetDeviceSelectors)
	if ok {
		if nf.IsRdma && !rdmaSpec.IsRdma() {
			glog.Warningf("RDMA resources for %s not found. Are RDMA modules loaded?", pciAddr)
		}
	}

	linkType := ""
	if len(ifName) > 0 {
		la, err := utils.GetLinkAttrs(ifName)
		if err != nil {
			return nil, err
		}
		linkType = la.EncapType
	}

	return &pciNetDevice{
		PciDevice: pciDev,
		ifName:    ifName,
		pfName:    pfName,
		pfAddr:    pfAddr,
		linkSpeed: "", // TO-DO: Get this using utils pkg
		rdmaSpec:  rdmaSpec,
		linkType:  linkType,
	}, nil
}

func (nd *pciNetDevice) GetPFName() string {
	return nd.pfName
}

func (nd *pciNetDevice) GetPFAddr() string {
	return nd.pfAddr
}

func (nd *pciNetDevice) GetNetName() string {
	return nd.ifName
}

func (nd *pciNetDevice) GetLinkSpeed() string {
	return nd.linkSpeed
}

func (nd *pciNetDevice) GetRdmaSpec() types.RdmaSpec {
	return nd.rdmaSpec
}

func (nd *pciNetDevice) GetLinkType() string {
	return nd.linkType
}

func (nd *pciNetDevice) GetDDPProfiles() string {
	pciAddr := nd.GetPciAddr()
	ddpProfile, err := utils.GetDDPProfiles(pciAddr)
	if err != nil {
		glog.Infof("GetDDPProfiles(): unable to get ddp profiles for device %s : %q", pciAddr, err)
		return ""
	}
	return ddpProfile
}
