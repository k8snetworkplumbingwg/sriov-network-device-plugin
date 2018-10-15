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

package resources

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

type resourceFactory struct {
	endPointPrefix string
	endPointSuffix string
}

var instance *resourceFactory

// NewResourceFactory returns an instance of Resource Server factory
func NewResourceFactory(prefix, suffix string) types.ResourceFactory {

	if instance == nil {
		return &resourceFactory{
			endPointPrefix: prefix,
			endPointSuffix: suffix,
		}
	}
	return instance
}

// GetResourceServer returns an instance of ResourceServer for a ResourcePool
func (rf *resourceFactory) GetResourceServer(rp types.ResourcePool) (types.ResourceServer, error) {
	if rp != nil {
		return newResourceServer(rf.endPointPrefix, rf.endPointSuffix, rp), nil
	}
	return nil, fmt.Errorf("factory: unable to get resource pool object")
}

// GetResourcePool returns and instance of ResourcePool for a ResourceConfig
func (rf *resourceFactory) GetResourcePool(rc *types.ResourceConfig) types.ResourcePool {
	glog.Infof("Resource pool type: %s", rc.DeviceType)
	switch rc.DeviceType {
	case "vfio":
		return newVfioResourcePool(rc)
	case "uio":
		return newUioResourcePool(rc)
	case "netdevice":
		return newNetDevicePool(rc)
	default:
		return newGenericResourcePool(rc)
	}
}
