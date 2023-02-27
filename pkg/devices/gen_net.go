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
	"fmt"

	"github.com/golang/glog"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

// GenNetDevice is a generic network device embedded into top level devices
type GenNetDevice struct {
	pfName    string
	pfAddr    string
	ifName    string
	linkType  string
	linkSpeed string
	funcID    int
	isRdma    bool
}

// NewGenNetDevice returns GenNetDevice instance
func NewGenNetDevice(deviceID string, dt types.DeviceType, isRdma bool) (*GenNetDevice, error) {
	var netNames []string
	var netNamesFromNs []string
	var pfName string
	var pfAddr string
	var funcID int
	var err error

	//nolint: exhaustive
	switch dt {
	case types.NetDeviceType:
		if pfName, err = utils.GetPfName(deviceID); err != nil {
			glog.Warningf("unable to get PF name %q", err.Error())
		}
		if pfAddr, err = utils.GetPfAddr(deviceID); err != nil {
			return nil, err
		}
		if funcID, err = utils.GetVFID(deviceID); err != nil {
			return nil, err
		}
		netNames, _ = utils.GetNetNames(deviceID)
		netNamesFromNs, _ = utils.GetNetNamesFromNetns(deviceID)
	case types.AuxNetDeviceType:
		if pfName, err = utils.GetSriovnetProvider().GetUplinkRepresentorFromAux(deviceID); err != nil {
			// AuxNetDeviceType by design should have PF, return error if failed to get PF name
			return nil, err
		}
		if pfAddr, err = utils.GetSriovnetProvider().GetPfPciFromAux(deviceID); err != nil {
			return nil, err
		}
		// Only SF auxiliary devices can have an index, for other (-1, err) returned.
		// TODO review this check in the future if other auxiliary device types are added
		if funcID, err = utils.GetSriovnetProvider().GetSfIndexByAuxDev(deviceID); err != nil {
			return nil, err
		}
		netNames, _ = utils.GetSriovnetProvider().GetNetDevicesFromAux(deviceID)
	default:
		return nil, fmt.Errorf("generic netdevices not supported for type %s", dt)
	}

	ifName := ""
	linkType := ""
	if len(netNames) > 0 {
		ifName = netNames[0]
		if len(ifName) > 0 {
			la, err := utils.GetNetlinkProvider().GetLinkAttrs(ifName)
			if err != nil {
				return nil, err
			}
			linkType = la.EncapType
		}
	} else if len(netNamesFromNs) > 0 {
		ifName = netNamesFromNs[0]
	}

	return &GenNetDevice{
		pfName:    pfName,
		pfAddr:    pfAddr,
		ifName:    ifName,
		linkType:  linkType,
		linkSpeed: "", // TODO: Get this using utils pkg
		funcID:    funcID,
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

// GetFuncID returns ID of the function
func (nd *GenNetDevice) GetFuncID() int {
	return nd.funcID
}

// IsRdma returns
func (nd *GenNetDevice) IsRdma() bool {
	return nd.isRdma
}
