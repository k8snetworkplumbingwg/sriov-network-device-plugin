// Copyright 2026 Hewlett Packard Enterprise. All Rights Reserved.
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
	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type cxiSpec struct {
	pciAddr string
}

// NewCxiSpec returns the CxiSpec
func NewCxiSpec(pciAddr string) types.CxiSpec {
	return &cxiSpec{pciAddr: pciAddr}
}

func (c *cxiSpec) IsCxi() bool {
	return utils.HasCxiDevice(c.pciAddr)
}

func (c *cxiSpec) GetCxiDeviceSpec() []*pluginapi.DeviceSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0, 1)

	cxiDev, err := utils.GetCxiDeviceFile(c.pciAddr)
	if err != nil {
		glog.Errorf("GetCxiDeviceSpec(): error getting cxi device file for device: %s, %v", c.pciAddr, err)
		return deviceSpec
	}

	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      cxiDev,
		ContainerPath: cxiDev,
		Permissions:   "rw",
	})

	return deviceSpec
}
