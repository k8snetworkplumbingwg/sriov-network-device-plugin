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
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

/*
	vfioResourcePool extends resourcePool and overrides:
	GetDeviceFile(),
	GetEnvs()
	GetMounts()
*/
type vfioResourcePool struct {
	vfioMount string
}

func newVfioResourcePool(rc *types.ResourceConfig) types.ResourcePool {
	return &resourcePool{
		config:        rc,
		devices:       make(map[string]*pluginapi.Device),
		deviceFiles:   make(map[string]string),
		IBaseResource: &vfioResourcePool{vfioMount: "/dev/vfio/vfio"},
	}
}

// Overrides GetDeviceFile() method
func (rp *vfioResourcePool) GetDeviceFile(dev string) (devFile string, err error) {
	// Get iommu group for this device
	devPath := filepath.Join(sysBusPci, dev)
	_, err = os.Lstat(devPath)
	if err != nil {
		err = fmt.Errorf("Error. Could not get directory information for device: %s, Err: %v", dev, err)
		return
	}

	iommuDir := filepath.Join(devPath, "iommu_group")
	if err != nil {
		err = fmt.Errorf("error reading iommuDir %v", err)
		return
	}

	dirInfo, err := os.Lstat(iommuDir)
	if err != nil {
		err = fmt.Errorf("unable to find iommu_group %v", err)
		return
	}

	if dirInfo.Mode()&os.ModeSymlink == 0 {
		err = fmt.Errorf("invalid symlink to iommu_group %v", err)
		return
	}

	linkName, err := filepath.EvalSymlinks(iommuDir)
	if err != nil {
		err = fmt.Errorf("error reading symlink to iommu_group %v", err)
		return
	}

	devFile = filepath.Join("/dev/vfio", filepath.Base(linkName))

	return
}

func (rp *vfioResourcePool) GetEnvs(resourceName string, deviceIDs []string) map[string]string {
	glog.Infof("vfio GetEnvs() called")
	envs := make(map[string]string)
	values := ""
	lastIndex := len(deviceIDs) - 1
	for i, s := range deviceIDs {
		values += s
		if i == lastIndex {
			break
		}
		values += " "
	}
	envs[resourceName] = values
	return envs
}

func (rp *vfioResourcePool) GetMounts() []*pluginapi.Mount {
	glog.Infof("vfio GetMounts() called")
	mounts := make([]*pluginapi.Mount, 0)
	return mounts
}

func (rp *vfioResourcePool) GetDeviceSpecs(deviceFiles map[string]string, deviceIDs []string) []*pluginapi.DeviceSpec {
	glog.Infof("vfio GetDeviceSpecs() called")
	devSpecs := make([]*pluginapi.DeviceSpec, 0)
	// Add default common vfio device file
	devSpecs = append(devSpecs, &pluginapi.DeviceSpec{
		HostPath:      rp.vfioMount,
		ContainerPath: rp.vfioMount,
		Permissions:   "mrw",
	})

	// Add vfio group specific devices
	for _, id := range deviceIDs {
		deviceFile := deviceFiles[id]
		ds := &pluginapi.DeviceSpec{
			HostPath:      deviceFile,
			ContainerPath: deviceFile,
			Permissions:   "mrw",
		}
		devSpecs = append(devSpecs, ds)
	}
	return devSpecs
}

// Probe returns 'true' if device health changes 'false' otherwise
func (rp *vfioResourcePool) Probe(rc *types.ResourceConfig, devices map[string]*pluginapi.Device) bool {
	return false
}
