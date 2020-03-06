// Copyright 2020 Red Hat, Inc. All Rights Reserved.
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
	"fmt"
	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

// netPoolInfo implements PoolDevice interface for Network Devices
type netPoolInfo struct {
	*resources.PoolInfoImpl
}

// newNetPoolInfo creates a netPoolInfo.
func newNetPoolInfo(netDev types.PciNetDevice, rc *types.ResourceConfig, rf types.ResourceFactory) (*netPoolInfo, error) {

	glog.Infof("Creating netPoolInfo for PciNetDevice: %+v\n", netDev.GetPciAddr())
	infoProvider := rf.GetInfoProvider(netDev.GetDriver())
	poolInfoImpl, err := resources.NewPoolInfoImpl(netDev, rc, infoProvider)
	if err != nil {
		return nil, err
	}

	// Append Rdma Specs only if Rdma is in the pool's selectors
	s, _ := rc.SelectorObj.(*types.NetDeviceSelectors)
	if s.IsRdma {
		rdmaSpec := netDev.GetRdmaSpec()
		if rdmaSpec.IsRdma() {
			rdmaDeviceSpec := rdmaSpec.GetRdmaDeviceSpec()
			for _, spec := range rdmaDeviceSpec {
				poolInfoImpl.DeviceSpecs = append(poolInfoImpl.DeviceSpecs, spec)
			}
		} else {
			return nil, fmt.Errorf("NewNetPoolInfo(): rdma is required in the configuration but the device %v is not an rdma device", netDev.GetPciAddr())
		}
	}

	return &netPoolInfo{
		poolInfoImpl,
	}, nil
}
