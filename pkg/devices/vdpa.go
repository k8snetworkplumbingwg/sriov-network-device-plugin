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

package devices

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type vdpaDevice struct {
	kvdpa.VdpaDevice
}

// GetType returns the VdpaType associated with the VdpaDevice
func (v *vdpaDevice) GetType() types.VdpaType {
	currentDriver := v.VdpaDevice.Driver()
	for vtype, driver := range types.SupportedVdpaTypes {
		if driver == currentDriver {
			return vtype
		}
	}
	return types.VdpaInvalidType
}

func (v *vdpaDevice) GetParent() string {
	return v.VdpaDevice.Name()
}

func (v *vdpaDevice) GetPath() (string, error) {
	if v.VhostVdpa() != nil {
		return v.VhostVdpa().Path(), nil
	}
	return "", fmt.Errorf("device %s: path is not applicable for non vhost-vdpa", v.Name())
}

// GetVdpaDevice returns a VdpaDevice from a given VF PCI address
func GetVdpaDevice(pciAddr string) types.VdpaDevice {
	detailVdpaDev, err := utils.GetVdpaProvider().GetVdpaDeviceByPci(pciAddr)
	if err != nil {
		glog.Infof("%s - No vDPA device found: %s", pciAddr, err)
		return nil
	}
	return &vdpaDevice{
		detailVdpaDev,
	}
}
