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

package infoprovider

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

const (
	legacyVFIOControl = "/dev/vfio/vfio"
	iommuControl      = "/dev/iommu"
)

type vfioInfoProvider struct {
	mu         sync.Mutex
	cdevPath   string // Stable for this provider lifetime; a remap requires rediscovery.
	pciAddr    string
	mode       string
	needIommu  bool
	hostDevDir string
}

// NewVfioInfoProvider creates a legacy VFIO provider.
func NewVfioInfoProvider(pciAddr string) types.DeviceInfoProvider {
	return NewVfioInfoProviderWithConfig(pciAddr, &types.ResourceConfig{})
}

// NewVfioInfoProviderWithConfig selects the VFIO interface for a resource pool.
func NewVfioInfoProviderWithConfig(pciAddr string, rc *types.ResourceConfig) types.DeviceInfoProvider {
	hostDevDir := rc.HostDevDir
	if hostDevDir == "" {
		hostDevDir = "/dev"
	}
	return &vfioInfoProvider{pciAddr: pciAddr, mode: rc.VFIOMode, needIommu: rc.NeedIommu, hostDevDir: hostDevDir}
}

func (rp *vfioInfoProvider) GetName() string { return "vfio" }

func (rp *vfioInfoProvider) GetDeviceSpecs() ([]*pluginapi.DeviceSpec, error) {
	if rp.mode == types.VFIOModeIOMMUFD {
		return rp.nativeDeviceSpecs()
	}
	specs := []*pluginapi.DeviceSpec{{HostPath: legacyVFIOControl, ContainerPath: legacyVFIOControl, Permissions: "rw"}}
	host, container, err := utils.GetVFIODeviceFile(rp.pciAddr)
	if err != nil {
		glog.Errorf("GetDeviceSpecs(): error getting legacy VFIO file for %s: %v", rp.pciAddr, err)
	} else {
		specs = append(specs, &pluginapi.DeviceSpec{HostPath: host, ContainerPath: container, Permissions: "rw"})
	}
	return specs, nil
}

func (rp *vfioInfoProvider) GetEnvVal() types.AdditionalInfo {
	if rp.mode == types.VFIOModeIOMMUFD {
		envs := types.AdditionalInfo{"mode": types.VFIOModeIOMMUFD}
		// Initialize on a metadata-first call. Retain the resolved identity on
		// failure; Allocate validates access before it obtains this metadata.
		_, _ = rp.GetDeviceSpecs()
		rp.mu.Lock()
		if rp.cdevPath != "" {
			envs["cdev-mount"] = rp.cdevPath
		}
		rp.mu.Unlock()
		if rp.needIommu {
			envs["iommu-mount"] = iommuControl
		}
		return envs
	}
	envs := types.AdditionalInfo{"mount": legacyVFIOControl}
	if _, container, err := utils.GetVFIODeviceFile(rp.pciAddr); err == nil {
		envs["dev-mount"] = container
	}
	return envs
}

func (rp *vfioInfoProvider) GetMounts() []*pluginapi.Mount { return nil }

// nativeDeviceSpecs validates access and pins the mapping to this discovery instance.
func (rp *vfioInfoProvider) nativeDeviceSpecs() ([]*pluginapi.DeviceSpec, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	path, err := utils.GetVFIOCdevFile(rp.pciAddr, rp.hostDevDir)
	if err != nil {
		return nil, fmt.Errorf("native VFIO device %s: %w", rp.pciAddr, err)
	}
	if rp.cdevPath != "" && path != rp.cdevPath {
		return nil, fmt.Errorf("native VFIO device %s changed cdev from %s to %s; rediscover the device", rp.pciAddr, rp.cdevPath, path)
	}
	specs := []*pluginapi.DeviceSpec{{HostPath: path, ContainerPath: path, Permissions: "rw"}}
	if rp.needIommu {
		if err := utils.ValidateIOMMUDevice(rp.hostDevDir); err != nil {
			return nil, fmt.Errorf("native VFIO device %s: %w", rp.pciAddr, err)
		}
		specs = append(specs, &pluginapi.DeviceSpec{HostPath: iommuControl, ContainerPath: iommuControl, Permissions: "rw"})
	}
	rp.cdevPath = path
	return specs, nil
}
