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

package infoprovider

import (
	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

/*
vfioInfoProvider implements DeviceInfoProvider
*/
type vfioInfoProvider struct {
	pciAddr          string
	vfioMount        string
	vfioDevHost      string
	vfioDevContainer string
}

// NewVfioInfoProvider create instance of VFIO DeviceInfoProvider
func NewVfioInfoProvider(pciAddr string) types.DeviceInfoProvider {
	vip := &vfioInfoProvider{
		pciAddr:   pciAddr,
		vfioMount: "/dev/vfio/vfio",
	}

	// Pre-compute VFIO device file so it's available for both GetDeviceSpecs and GetEnvVal
	// regardless of call order
	vfioDevHost, vfioDevContainer, err := utils.GetVFIODeviceFile(pciAddr)
	if err != nil {
		glog.Errorf("NewVfioInfoProvider(): error getting vfio device file for device: %s, %s", pciAddr, err.Error())
	} else {
		vip.vfioDevHost = vfioDevHost
		vip.vfioDevContainer = vfioDevContainer
	}

	return vip
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (rp *vfioInfoProvider) GetName() string {
	return "vfio"
}

func (rp *vfioInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	devSpecs := make([]*pluginapi.DeviceSpec, 0)
	devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
		HostPath:      rp.vfioMount,
		ContainerPath: rp.vfioMount,
		Permissions:   "rw",
	})

	if rp.vfioDevHost != "" && rp.vfioDevContainer != "" {
		devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
			HostPath:      rp.vfioDevHost,
			ContainerPath: rp.vfioDevContainer,
			Permissions:   "rw",
		})
	}

	return devSpecs
}

func (rp *vfioInfoProvider) GetEnvVal() types.AdditionalInfo {
	envs := make(map[string]string, 0)
	envs["mount"] = "/dev/vfio/vfio"
	if rp.vfioDevContainer != "" {
		envs["dev-mount"] = rp.vfioDevContainer
	}

	return envs
}

func (rp *vfioInfoProvider) GetMounts() []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// *****************************************************************
