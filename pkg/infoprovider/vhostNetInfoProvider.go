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

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

var (
	// HostNet variable for vhost-net device path
	// used to check path for unit-tests
	HostNet = "/dev/vhost-net"
	// HostTun variable for tun device path
	// used to check path for unit-tests
	HostTun = "/dev/net/tun"
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
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      "/dev/vhost-net",
		ContainerPath: "/dev/vhost-net",
		Permissions:   "rw",
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
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      "/dev/net/tun",
		ContainerPath: "/dev/net/tun",
		Permissions:   "rw",
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
		glog.Errorf("GetDeviceSpecs(): /dev/vhost-net doesn't exist")
		return nil
	}
	deviceSpec := getVhostNetDeviceSpec()

	if !tunDeviceExist() {
		glog.Errorf("GetDeviceSpecs(): /dev/net/tun doesn't exist")
		return nil
	}
	deviceSpec = append(deviceSpec, getTunDeviceSpec()...)

	return deviceSpec
}

func (ip *vhostNetInfoProvider) GetEnvVal() types.AdditionalInfo {
	envs := make(map[string]string, 0)
	envs["net-mount"] = "/dev/vhost-net"
	envs["tun-mount"] = "/dev/net/tun"

	return envs
}

func (ip *vhostNetInfoProvider) GetMounts() []*pluginapi.Mount {
	return nil
}

// *****************************************************************
