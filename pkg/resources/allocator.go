// Copyright 2022 Intel Corp. All Rights Reserved.
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
	"sort"

	"github.com/golang/glog"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

// DeviceSet is to hold and manipulate a set of HostDevice
type DeviceSet map[string]types.HostDevice

// PackedAllocator implements the Allocator interface
type PackedAllocator struct {
}

// NewPackedAllocator create instance of PackedAllocator
func NewPackedAllocator() *PackedAllocator {
	return &PackedAllocator{}
}

// NewDeviceSet is to create an empty DeviceSet
func NewDeviceSet() DeviceSet {
	set := make(DeviceSet)
	return set
}

// Insert is to add a HostDevice in DeviceSet
func (ds *DeviceSet) Insert(pciAddr string, device types.HostDevice) {
	(*ds)[pciAddr] = device
}

// Delete is to delete a HostDevice in DeviceSet
func (ds *DeviceSet) Delete(pciAddr string) {
	delete(*ds, pciAddr)
}

// AsSortedStrings is to sort the DeviceSet and return the sorted keys
func (ds *DeviceSet) AsSortedStrings() []string {
	keys := make([]string, 0, len(*ds))
	for k := range *ds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Allocate return the preferred allocation
func (pa *PackedAllocator) Allocate(rqt *pluginapi.ContainerPreferredAllocationRequest, rp types.ResourcePool) []string {
	size := rqt.AllocationSize
	preferredDevices := make([]string, 0)

	if size <= 0 {
		glog.Warningf("Allocator(): requested number of devices are negative. requested: %d", size)
		return []string{}
	}

	if len(rqt.AvailableDeviceIDs) < int(size) {
		glog.Warningf("Allocator(): not enough number of devices were available. available: %d, requested: %d", len(rqt.AvailableDeviceIDs), size)
		return []string{}
	}

	if len(rqt.MustIncludeDeviceIDs) > int(size) {
		glog.Warningf("Allocator(): allocated number of devices exceeded the number of requested devices. allocated: %d, requested: %d",
			len(rqt.MustIncludeDeviceIDs),
			size)
	}

	availableSet := NewDeviceSet()
	for _, available := range rqt.AvailableDeviceIDs {
		dev, ok := rp.GetDevicePool()[available]
		if ok {
			availableSet.Insert(available, dev)
		} else {
			glog.Warningf("Allocator(): not available device id was specified: %s", available)
			return []string{}
		}
	}
	for _, required := range rqt.MustIncludeDeviceIDs {
		_, ok := rp.GetDevicePool()[required]
		if ok {
			availableSet.Delete(required)
		} else {
			glog.Warningf("Allocator(): not available device was included: %s", required)
			return []string{}
		}
	}
	sortedAvailableSet := availableSet.AsSortedStrings()

	preferredDevices = append(preferredDevices, rqt.MustIncludeDeviceIDs...)
	if len(preferredDevices) < int(size) {
		preferredDevices = append(preferredDevices, sortedAvailableSet[:int(size)-len(preferredDevices)]...)
	}
	return preferredDevices
}
