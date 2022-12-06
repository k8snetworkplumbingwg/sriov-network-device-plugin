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

package accelerator

import (
	"github.com/jaypipes/ghw"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

// accelDevice extends HostDevice and embedds GenPciDevice
type accelDevice struct {
	types.HostDevice
	devices.GenPciDevice
}

// NewAccelDevice returns an instance of AccelDevice interface
func NewAccelDevice(dev *ghw.PCIDevice, rFactory types.ResourceFactory,
	rc *types.ResourceConfig) (types.AccelDevice, error) {
	hostDev, err := devices.NewHostDeviceImpl(dev, dev.Address, rFactory, rc, nil)
	if err != nil {
		return nil, err
	}
	pciDev, err := devices.NewGenPciDevice(dev)
	if err != nil {
		return nil, err
	}

	return &accelDevice{
		HostDevice:   hostDev,
		GenPciDevice: *pciDev,
	}, nil
}
