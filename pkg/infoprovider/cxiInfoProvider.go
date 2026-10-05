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

package infoprovider

import (
	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

/*
cxiInfoProvider provides the Cassini (CXI) char device information.
It mounts the /dev/cxiN char device associated with a CXI VF into the container.
*/
type cxiInfoProvider struct {
	cxiSpec types.CxiSpec
}

// NewCxiInfoProvider returns a new CXI Information Provider
func NewCxiInfoProvider(cxiSpec types.CxiSpec) types.DeviceInfoProvider {
	return &cxiInfoProvider{
		cxiSpec: cxiSpec,
	}
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (ip *cxiInfoProvider) GetName() string {
	return "cxi"
}

func (ip *cxiInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if !ip.cxiSpec.IsCxi() {
		glog.Errorf("GetDeviceSpecs(): cxi is required in the configuration but the device is not cxi device")
		return nil
	}

	return ip.cxiSpec.GetCxiDeviceSpec()
}

func (ip *cxiInfoProvider) GetEnvVal() types.AdditionalInfo {
	envs := make(map[string]string, 0)
	for _, devSpec := range ip.cxiSpec.GetCxiDeviceSpec() {
		envs["dev-mount"] = devSpec.ContainerPath
	}

	return envs
}

func (ip *cxiInfoProvider) GetMounts() []*pluginapi.Mount {
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

// *****************************************************************
