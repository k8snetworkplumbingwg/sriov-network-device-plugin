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

package infoprovider

import (
	"strings"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

/*
rdmaInfoProvider provides the RDMA information
*/
type rdmaInfoProvider struct {
	rdmaSpec types.RdmaSpec
}

// NewRdmaInfoProvider returns a new Rdma Information Provider
func NewRdmaInfoProvider(rdmaSpec types.RdmaSpec) types.DeviceInfoProvider {
	return &rdmaInfoProvider{
		rdmaSpec: rdmaSpec,
	}
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (ip *rdmaInfoProvider) GetName() string {
	return "rdma"
}

func (ip *rdmaInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if !ip.rdmaSpec.IsRdma() {
		glog.Errorf("GetDeviceSpecs(): rdma is required in the configuration but the device is not rdma device")
		return nil
	}

	devsSpec := ip.rdmaSpec.GetRdmaDeviceSpec()
	return devsSpec
}

func (ip *rdmaInfoProvider) GetEnvVal() types.AdditionalInfo {
	envs := make(map[string]string, 0)
	devsSpec := ip.rdmaSpec.GetRdmaDeviceSpec()
	for _, devSpec := range devsSpec {
		if strings.Contains(devSpec.ContainerPath, "uverbs") {
			envs["uverbs"] = devSpec.ContainerPath
		} else if strings.Contains(devSpec.ContainerPath, "umad") {
			envs["umad"] = devSpec.ContainerPath
		} else if strings.Contains(devSpec.ContainerPath, "issm") {
			envs["issm"] = devSpec.ContainerPath
		} else if strings.Contains(devSpec.ContainerPath, "rdma_cm") {
			envs["rdma_cm"] = devSpec.ContainerPath
		}
	}

	return envs
}

func (ip *rdmaInfoProvider) GetMounts() []*pluginapi.Mount {
	return nil
}

// *****************************************************************
