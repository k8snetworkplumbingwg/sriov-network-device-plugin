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

package netdevice

import (
	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type netResourcePool struct {
	*resources.ResourcePoolImpl
	selectors *types.NetDeviceSelectors
}

var _ types.ResourcePool = &netResourcePool{}

// NewNetResourcePool returns an instance of resourcePool
func NewNetResourcePool(rc *types.ResourceConfig, filteredDevice []types.PciDevice, rf types.ResourceFactory) (types.ResourcePool, error) {
	poolInfoMap := make(map[string]types.PoolInfo, 0)
	apiDevices := make(map[string]*pluginapi.Device)
	for _, dev := range filteredDevice {
		pciAddr := dev.GetPciAddr()
		netDev, _ := dev.(types.PciNetDevice)
		poolInfo, err := newNetPoolInfo(netDev, rc, rf)
		if err != nil {
			glog.Errorf("Failed to obtain Pool Information for device: [pciAddr: %s, vendor: %s, device: %s, driver: %s]",
				dev.GetPciAddr(),
				dev.GetVendor(),
				dev.GetDeviceCode(),
				dev.GetDriver())
			return nil, err
		}
		poolInfoMap[pciAddr] = poolInfo
		apiDevices[pciAddr] = poolInfo.GetAPIDevice()
		glog.Infof("device added: [pciAddr: %s, vendor: %s, device: %s, driver: %s]",
			dev.GetPciAddr(),
			dev.GetVendor(),
			dev.GetDeviceCode(),
			dev.GetDriver())
	}

	rp := resources.NewResourcePool(rc, apiDevices, poolInfoMap)
	s, _ := rc.SelectorObj.(*types.NetDeviceSelectors)
	return &netResourcePool{
		ResourcePoolImpl: rp,
		selectors:        s,
	}, nil
}
