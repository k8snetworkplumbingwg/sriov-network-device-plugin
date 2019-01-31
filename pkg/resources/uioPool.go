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
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type uioResourcePool struct {
}

func newUioResourcePool() types.DeviceInfoProvider {
	return &uioResourcePool{}
}

// *****************************************************************
/* DeviceInfoProvider Interface */
func (rp *uioResourcePool) GetDeviceSpecs(pciAddr string) []*pluginapi.DeviceSpec {
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	uioDev, err := utils.GetUIODeviceFile(pciAddr)
	if err != nil {
		glog.Errorf("GetDeviceSpecs(): error getting vfio device file for device: %s", pciAddr)
	} else {
		devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
			HostPath:      uioDev,
			ContainerPath: uioDev,
			Permissions:   "mrw",
		})
	}

	return devSpecs
}

func (rp *uioResourcePool) GetEnvVal(pciAddr string) string {
	return pciAddr
}

func (rp *uioResourcePool) GetMounts(pciAddr string) []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// *****************************************************************
