/*
Copyright 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package netdevice

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

func (rip *rdmaInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if !rip.rdmaSpec.IsRdma() {
		glog.Errorf("GetDeviceSpecs(): rdma is required in the configuration but the device is not rdma device")
		return nil
	}
	return rip.rdmaSpec.GetRdmaDeviceSpec()
}

func (rip *rdmaInfoProvider) GetEnvVal() string {
	return ""
}

func (rip *rdmaInfoProvider) GetMounts() []*pluginapi.Mount {
	return nil
}

/*
VhostNetInfoProvider wraps any DeviceInfoProvider and adds a vhost-net device
*/
type vhostNetInfoProvider struct {
}

// NewVhostNetInfoProvider returns a new Vhost Information Provider
func NewVhostNetInfoProvider() types.DeviceInfoProvider {
	return &vhostNetInfoProvider{}
}

// *****************************************************************
/* DeviceInfoProvider Interface */

func (rip *vhostNetInfoProvider) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	if !VhostNetDeviceExist() {
		glog.Errorf("GetDeviceSpecs(): /dev/vhost-net doesn't exist")
		return nil
	}
	deviceSpec := GetVhostNetDeviceSpec()

	if !TunDeviceExist() {
		glog.Errorf("GetDeviceSpecs(): /dev/net/tun doesn't exist")
		return nil
	}
	deviceSpec = append(deviceSpec, GetTunDeviceSpec()...)

	return deviceSpec
}

func (rip *vhostNetInfoProvider) GetEnvVal() string {
	return ""
}

func (rip *vhostNetInfoProvider) GetMounts() []*pluginapi.Mount {
	return nil
}
