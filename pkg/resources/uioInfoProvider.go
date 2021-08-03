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
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type uioInfoProvider struct {
	pciAddr string
}

// NewUioInfoProvider return instance of uio DeviceInfoProvider
func NewUioInfoProvider(pciAddr string) types.DeviceInfoProvider {
	return &uioInfoProvider{
		pciAddr: pciAddr,
	}
}

// *****************************************************************
/* DeviceInfoProvider Interface */
func (rp *uioInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	uioDev, err := utils.GetUIODeviceFile(rp.pciAddr)
	if err != nil {
		glog.Errorf("GetDeviceSpecs(): error getting vfio device file for device: %s", rp.pciAddr)
	} else {
		devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
			HostPath:      uioDev,
			ContainerPath: uioDev,
			Permissions:   "mrw",
		})
	}

	return devSpecs
}

func (rp *uioInfoProvider) GetEnvVal() string {
	return rp.pciAddr
}

func (rp *uioInfoProvider) GetMounts() []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// *****************************************************************
