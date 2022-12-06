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

func (ip *rdmaInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if !ip.rdmaSpec.IsRdma() {
		glog.Errorf("GetDeviceSpecs(): rdma is required in the configuration but the device is not rdma device")
		return nil
	}
	return ip.rdmaSpec.GetRdmaDeviceSpec()
}

func (ip *rdmaInfoProvider) GetEnvVal() string {
	return ""
}

func (ip *rdmaInfoProvider) GetMounts() []*pluginapi.Mount {
	return nil
}
