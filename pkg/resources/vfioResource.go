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

package resources

import (
	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

/*
	vfioResource extends resourcePool and overrides:
	GetDeviceSpecs(),
	GetEnvs()
	GetMounts()
*/
type vfioResource struct {
	vfioMount string
}

// NewVfioResource create instance of VFIO DeviceInfoProvider
func NewVfioResource() types.DeviceInfoProvider {

	return &vfioResource{
		vfioMount: "/dev/vfio/vfio",
	}

}

// *****************************************************************
/* DeviceInfoProvider Interface */
func (rp *vfioResource) GetDeviceSpecs(pciAddr string) []*pluginapi.DeviceSpec {
	devSpecs := make([]*pluginapi.DeviceSpec, 0)
	devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
		HostPath:      rp.vfioMount,
		ContainerPath: rp.vfioMount,
		Permissions:   "mrw",
	})

	vfioDevHost, vfioDevContainer, err := utils.GetVFIODeviceFile(pciAddr)
	if err != nil {
		glog.Errorf("GetDeviceSpecs(): error getting vfio device file for device: %s, %s", pciAddr, err.Error())
	} else {
		devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
			HostPath:      vfioDevHost,
			ContainerPath: vfioDevContainer,
			Permissions:   "mrw",
		})
	}

	return devSpecs
}

func (rp *vfioResource) GetEnvVal(pciAddr string) string {
	return pciAddr
}

func (rp *vfioResource) GetMounts(pciAddr string) []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// *****************************************************************
