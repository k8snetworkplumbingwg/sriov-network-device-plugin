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

package netdevice

import (
	"github.com/golang/glog"
	"github.com/jaypipes/ghw"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

func newAuxNetDevice(dev *ghw.PCIDevice, auxDev string, rFactory types.ResourceFactory, rc *types.ResourceConfig) (types.PciNetDevice, error) {
	infoProviders := make([]types.DeviceInfoProvider, 0)

	driverName, err := utils.GetDriverName(dev.Address)
	if err != nil {
		return nil, err
	}

	infoProviders = append(infoProviders, rFactory.GetDefaultInfoProvider(auxDev, driverName))

	var rdmaSpec types.RdmaSpec
	nf, ok := rc.SelectorObj.(*types.NetDeviceSelectors)
	if ok {
		// Add InfoProviders based on Selector data
		if nf.IsRdma {
			rdmaSpec = NewAuxRdmaSpec(auxDev)
			if rdmaSpec.IsRdma() {
				infoProviders = append(infoProviders, NewRdmaInfoProvider(rdmaSpec))
			} else {
				glog.Warningf("RDMA resources for %s not found. Are RDMA modules loaded?", auxDev)
			}
		}
	}

	pciDev, err := resources.NewPciDevice(dev, rFactory, rc, infoProviders)
	if err != nil {
		return nil, err
	}

	// that's a hack to be able to use auxiliary ID as device ID
	apiDevice := pciDev.GetAPIDevice()
	apiDevice.ID = auxDev

	pfName, err := utils.GetUplinkRepresentorFromAuxDev(auxDev)
	if err != nil {
		glog.Warningf("unable to get PF name %q for device %s", err.Error(), auxDev)
	}

	linkType := ""
	ifName, _ := utils.GetAuxDevIfName(auxDev)
	if len(ifName) > 0 {
		la, err := utils.GetNetlinkProvider().GetLinkAttrs(ifName)
		if err != nil {
			return nil, err
		}
		linkType = la.EncapType
	}

	return &pciNetDevice{
		PciDevice: pciDev,
		ifName:    ifName,
		pfName:    pfName,
		linkSpeed: "", // TODO: Get this using utils pkg
		rdmaSpec:  rdmaSpec,
		linkType:  linkType,
	}, nil
}

// NewAuxNetDevices creates auxiliary devices for specified PCI device
func NewAuxNetDevices(dev *ghw.PCIDevice, rFactory types.ResourceFactory, rc *types.ResourceConfig) ([]types.PciDevice, error) {
	newPciDevices := make([]types.PciDevice, 0)
	auxDevs, err := utils.GetAuxNetDevicesFromPci(dev.Address)
	if err != nil {
		return nil, err
	}
	for _, auxDev := range auxDevs {
		if newDevice, err := newAuxNetDevice(dev, auxDev, rFactory, rc); err == nil {
			newPciDevices = append(newPciDevices, newDevice)
		} else {
			glog.Errorf("netdevice NewAuxNetDevices(): error creating new auxiliary device: %q", err)
		}
	}
	return newPciDevices, nil
}
