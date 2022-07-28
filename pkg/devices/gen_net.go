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
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

// GenNetDevice is a generic network device embedded into top level devices
type GenNetDevice struct {
	pfName    string
	pfAddr    string
	ifName    string
	linkType  string
	linkSpeed string
	vfID      int
	isRdma    bool
}

// NewGenNetDevice returns GenNetDevice instance
func NewGenNetDevice(dev *ghw.PCIDevice, isRdma bool) (*GenNetDevice, error) {
	var ifName string
	pfName, err := utils.GetPfName(dev.Address)
	if err != nil {
		glog.Warningf("unable to get PF name %q", err.Error())
	}

	netDevs, _ := utils.GetNetNames(dev.Address)
	if len(netDevs) == 0 {
		ifName = ""
	} else {
		ifName = netDevs[0]
	}

	// Get PF PCI address
	pfAddr, err := utils.GetPfAddr(dev.Address)
	if err != nil {
		return nil, err
	}

	linkType := ""
	if len(ifName) > 0 {
		la, err := utils.GetNetlinkProvider().GetLinkAttrs(ifName)
		if err != nil {
			return nil, err
		}
		linkType = la.EncapType
	}

	vfID, err := utils.GetVFID(dev.Address)
	if err != nil {
		return nil, err
	}

	return &GenNetDevice{
		pfName:    pfName,
		pfAddr:    pfAddr,
		ifName:    ifName,
		linkType:  linkType,
		linkSpeed: "", // TODO: Get this using utils pkg
		vfID:      vfID,
		isRdma:    isRdma,
	}, nil
}

// GetPfNetName returns PF netdevice name
func (nd *GenNetDevice) GetPfNetName() string {
	return nd.pfName
}

// GetPfPciAddr returns PF pci address
func (nd *GenNetDevice) GetPfPciAddr() string {
	return nd.pfAddr
}

// GetNetName returns name of the network interface
func (nd *GenNetDevice) GetNetName() string {
	return nd.ifName
}

// GetLinkSpeed returns link speed
func (nd *GenNetDevice) GetLinkSpeed() string {
	return nd.linkSpeed
}

// GetLinkType returns link type
func (nd *GenNetDevice) GetLinkType() string {
	return nd.linkType
}

// GetVFID returns ID of the VF
func (nd *GenNetDevice) GetVFID() int {
	return nd.vfID
}

// IsRdma returns
func (nd *GenNetDevice) IsRdma() bool {
	return nd.isRdma
}
