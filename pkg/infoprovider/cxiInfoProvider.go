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
cxiInfoProvider provides the Cassini (CXI) char device information.
It mounts the /dev/cxiN char device associated with a CXI VF into the container.
*/
type cxiInfoProvider struct {
	pciAddr string
}

// NewCxiInfoProvider returns a new CXI Information Provider
func NewCxiInfoProvider(pciAddr string) types.DeviceInfoProvider {
	return &cxiInfoProvider{
		pciAddr: pciAddr,
	}
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (ip *cxiInfoProvider) GetName() string {
	return "cxi"
}

func (ip *cxiInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	cxiDev, err := utils.GetCxiDeviceFile(ip.pciAddr)
	if err != nil {
		glog.Errorf("GetDeviceSpecs(): error getting cxi device file for device: %s, %s", ip.pciAddr, err.Error())
		return devSpecs
	}

	// Note: the existence of the /dev char device is intentionally not verified here.
	// HostPath is resolved by kubelet on the host, whereas this code runs inside the
	// device-plugin container which does not mount the host's /dev.

	devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
		HostPath:      cxiDev,
		ContainerPath: cxiDev,
		Permissions:   "rw",
	})

	return devSpecs
}

func (ip *cxiInfoProvider) GetEnvVal() types.AdditionalInfo {
	envs := make(map[string]string, 0)

	cxiDev, err := utils.GetCxiDeviceFile(ip.pciAddr)
	if err != nil {
		glog.Errorf("GetEnvVal(): error getting cxi device file for device: %s, %s", ip.pciAddr, err.Error())
	} else {
		envs["dev-mount"] = cxiDev
	}

	return envs
}

func (ip *cxiInfoProvider) GetMounts() []*pluginapi.Mount {
	return nil
}

// *****************************************************************
