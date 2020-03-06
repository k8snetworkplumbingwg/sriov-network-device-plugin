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

package accelerator

import (
	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

// accelPoolInfo implements PoolDevice interface for Accelerator Devices
type accelPoolInfo struct {
	*resources.PoolInfoImpl
}

// newAccelPoolInfo create a accelPoolInfo
func newAccelPoolInfo(accelDev types.AccelDevice, rc *types.ResourceConfig, rf types.ResourceFactory) (*accelPoolInfo, error) {

	glog.Infof("Creating accelPoolInfo for AccelDevice: %+v\n", accelDev.GetPciAddr())
	infoProvider := rf.GetInfoProvider(accelDev.GetDriver())
	poolInfoImpl, err := resources.NewPoolInfoImpl(accelDev, rc, infoProvider)
	if err != nil {
		return nil, err
	}
	return &accelPoolInfo{
		poolInfoImpl,
	}, nil
}
