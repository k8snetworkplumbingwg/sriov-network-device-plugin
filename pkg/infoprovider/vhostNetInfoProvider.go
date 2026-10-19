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
	"os"

	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

const (
	// vhostNetPath is the real device path for vhost-net
	vhostNetPath = "/dev/vhost-net"
	// tunPath is the real device path for tun
	tunPath = "/dev/net/tun"
)

var (
	// HostNet variable for vhost-net device path
	// used to check path for unit-tests
	HostNet = vhostNetPath
	// HostTun variable for tun device path
	// used to check path for unit-tests
	HostTun = tunPath
)

/*
VhostNetInfoProvider wraps any DeviceInfoProvider and adds a vhost-net device
*/
type vhostNetInfoProvider struct {
}

// NewVhostNetInfoProvider returns a new Vhost Information Provider
func NewVhostNetInfoProvider() types.DeviceInfoProvider {
	return &vhostNetInfoProvider{}
}

// VhostNetDeviceExist returns true if /dev/vhost-net exists
func VhostNetDeviceExist() bool {
	_, err := os.Stat(HostNet)
	return err == nil
}

// GetVhostNetDeviceSpec returns an instance of DeviceSpec for vhost-net
func getVhostNetDeviceSpec() []*pluginapi.DeviceSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0, 1)
	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      vhostNetPath,
		ContainerPath: vhostNetPath,
		Permissions:   devPermissions,
	})

	return deviceSpec
}

// TunDeviceExist returns true if /dev/net/tun exists
func tunDeviceExist() bool {
	_, err := os.Stat(HostTun)
	return err == nil
}

// GetTunDeviceSpec returns an instance of DeviceSpec for Tun
func getTunDeviceSpec() []*pluginapi.DeviceSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0, 1)
	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      tunPath,
		ContainerPath: tunPath,
		Permissions:   devPermissions,
	})

	return deviceSpec
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (ip *vhostNetInfoProvider) GetName() string {
	return "vhost"
}

func (ip *vhostNetInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if !VhostNetDeviceExist() {
		klog.Errorf("GetDeviceSpecs(): %s doesn't exist", vhostNetPath)
		return nil
	}
	deviceSpec := getVhostNetDeviceSpec()

	if !tunDeviceExist() {
		klog.Errorf("GetDeviceSpecs(): %s doesn't exist", tunPath)
		return nil
	}
	deviceSpec = append(deviceSpec, getTunDeviceSpec()...)

	return deviceSpec
}

func (ip *vhostNetInfoProvider) GetEnvVal() types.AdditionalInfo {
	envs := make(map[string]string, 0)
	envs["net-mount"] = vhostNetPath
	envs["tun-mount"] = tunPath

	return envs
}

func (ip *vhostNetInfoProvider) GetMounts() []*pluginapi.Mount {
	return nil
}

// *****************************************************************
