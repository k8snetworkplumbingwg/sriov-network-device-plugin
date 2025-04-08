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
	"fmt"
	"strings"

	"github.com/golang/glog"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type netResourcePool struct {
	*resources.ResourcePoolImpl
	nadutils types.NadUtils
	config   *types.ResourceConfig
}

var _ types.ResourcePool = &netResourcePool{}

// NewNetResourcePool returns an instance of resourcePool
func NewNetResourcePool(nadutils types.NadUtils, rc *types.ResourceConfig,
	devicePool map[string]types.HostDevice) types.ResourcePool {
	rp := resources.NewResourcePool(rc, devicePool)
	return &netResourcePool{
		ResourcePoolImpl: rp,
		nadutils:         nadutils,
		config:           rc,
	}
}

// Overrides GetDeviceSpecs
func (rp *netResourcePool) GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec {
	glog.Infof("GetDeviceSpecs(): for devices: %v", deviceIDs)
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	devicePool := rp.GetDevicePool()

	// Add device driver specific and rdma specific devices
	for _, id := range deviceIDs {
		if dev, ok := devicePool[id]; ok {
			netDev := dev.(types.PciNetDevice) // convert generic HostDevice to PciNetDevice
			newSpecs := netDev.GetDeviceSpecs()
			for _, ds := range newSpecs {
				if !rp.DeviceSpecExist(devSpecs, ds) {
					devSpecs = append(devSpecs, ds)
				}
			}
		}
	}
	return devSpecs
}

func (rp *netResourcePool) Probe() bool {
	// 1. If the physical function PF of the SR-IOV devices is carrier down. This should be marked unhealthy. Normally, SR-IOV
	// would still function when the PF is carrier down. But in the case of DPUs/IPUs/SmartNics with an embedded CPU,
	// the PF being down **can** signal that the embedded CPU is in reset or shutdown with carrier down.
	// 2. If any of the devices are gone. This could be due to someone changing the number of virtual functions.
	// Or in the case of DPUs/IPUs/SmartNics with an embedded CPU, the driver needed to reset. This will cause the
	// virtual functions to be removed. All devices that are gone should be marked unhealthy. Normally this won't be the case
	// since the SR-IOV Network Operator will be managing the SR-IOV devices. However for DPUs/IPUs/SmartNics with an embedded CPU,
	// would be externally managed with a separate operator.
	changes := false
	cachedPfLinkStatus := make(map[string]bool)
	for id, device := range rp.GetDevicePool() {
		netDev, ok := device.(types.PciNetDevice)
		if !ok {
			// Skip devices that are not PCI net devices
			continue
		}
		currentHealth := device.GetHealth()
		pfName := netDev.GetPfNetName()

		pfIsUp := true
		var err error
		pfIsUpLog := ""
		if rp.config.CheckHealthOnPf {
			if cachedStatus, exists := cachedPfLinkStatus[pfName]; exists {
				pfIsUp = cachedStatus
			} else {
				pfIsUp, err = netDev.IsPfLinkUp()
				if err != nil {
					// If we can't check the link status, assume it's up. It could be that the PF was moved to a different netns.
					// We want a conservative approach, as we don't want to mark the device as unhealthy if we are unsure.
					pfIsUp = true
				}
				cachedPfLinkStatus[pfName] = pfIsUp
			}
			pfIsUpLog = fmt.Sprintf("PF %s", pfName)
			if pfIsUp {
				pfIsUpLog = fmt.Sprintf("%s is UP, ", pfIsUpLog)
			} else {
				pfIsUpLog = fmt.Sprintf("%s is DOWN, ", pfIsUpLog)
			}
		}

		deviceExists := true
		deviceExistsLog := ""
		if rp.config.CheckHealthOnDeviceExist {
			deviceExists = netDev.DeviceExists()
			deviceExistsLog = fmt.Sprintf("Device %s", netDev.GetPciAddr())
			if deviceExists {
				deviceExistsLog = fmt.Sprintf("%s is existing, ", deviceExistsLog)
			} else {
				deviceExistsLog = fmt.Sprintf("%s is missing, ", deviceExistsLog)
			}
		}

		if pfIsUp && deviceExists && !currentHealth {
			glog.Infof("%s%sdevice was unhealthy, marking device %s as healthy", pfIsUpLog, deviceExistsLog, id)
			device.SetHealth(true)
			changes = true
		} else if !pfIsUp && deviceExists && currentHealth {
			glog.Infof("%s%sdevice was healthy, marking device %s as unhealthy", pfIsUpLog, deviceExistsLog, id)
			device.SetHealth(false)
			changes = true
		} else if pfIsUp && !deviceExists && currentHealth {
			glog.Infof("%s%sdevice was healthy, marking device %s as unhealthy", pfIsUpLog, deviceExistsLog, id)
			device.SetHealth(false)
			changes = true
		} else if !pfIsUp && !deviceExists && currentHealth {
			glog.Infof("%s%sdevice was healthy, marking device %s as unhealthy", pfIsUpLog, deviceExistsLog, id)
			device.SetHealth(false)
			changes = true
		}
	}
	return changes
}

// StoreDeviceInfoFile stores the Device Info files according to the
// k8snetworkplumbingwg/device-info-spec
// for the requested deviceIDs
func (rp *netResourcePool) StoreDeviceInfoFile(resourceNamePrefix string, deviceIDs []string) error {
	var devInfo nettypes.DeviceInfo
	devicePool := rp.GetDevicePool()

	for _, id := range deviceIDs {
		dev := devicePool[id]
		netDev, ok := dev.(types.PciNetDevice)
		if !ok {
			return fmt.Errorf("storeDeviceInfoFile: Only pciNetDevices are supported")
		}

		vdpaDev := netDev.GetVdpaDevice()
		if vdpaDev != nil {
			var vdpaDevice *nettypes.VdpaDevice = nil
			if vdpaDev.GetType() == types.VdpaVhostType {
				vdpaPath, err := vdpaDev.GetPath()
				if err == nil {
					vdpaDevice = &nettypes.VdpaDevice{
						ParentDevice: vdpaDev.GetParent(),
						Driver:       string(vdpaDev.GetType()),
						Path:         vdpaPath,
						PciAddress:   netDev.GetPciAddr(),
					}
				} else {
					glog.Errorf("Unexpected error when fetching the vdpa device path: %s", err)
				}
			}
			// either virtio/vDPA case or not able to get a mount path for vhost/vDPA
			if vdpaDevice == nil {
				vdpaDevice = &nettypes.VdpaDevice{
					ParentDevice: vdpaDev.GetParent(),
					Driver:       string(vdpaDev.GetType()),
					PciAddress:   netDev.GetPciAddr(),
				}
			}

			devInfo = nettypes.DeviceInfo{
				Type:    nettypes.DeviceInfoTypeVDPA,
				Version: nettypes.DeviceInfoVersion,
				Vdpa:    vdpaDevice,
			}
		} else {
			devInfo = nettypes.DeviceInfo{
				Type:    nettypes.DeviceInfoTypePCI,
				Version: nettypes.DeviceInfoVersion,
				Pci: &nettypes.PciDevice{
					PciAddress: netDev.GetPciAddr(),
				},
			}

			if netDev.IsRdma() {
				rdmaDevices := utils.GetRdmaProvider().GetRdmaDevicesForPcidev(devInfo.Pci.PciAddress)
				if len(rdmaDevices) == 0 {
					glog.Errorf("No RDMA devices available for RDMA capable device: %s", devInfo.Pci.PciAddress)
				} else {
					devInfo.Pci.RdmaDevice = strings.Join(rdmaDevices, ",")
				}
			}
		}
		resource := fmt.Sprintf("%s/%s", resourceNamePrefix, rp.GetConfig().ResourceName)
		if err := rp.nadutils.CleanDeviceInfoFile(resource, id); err != nil {
			return err
		}
		if err := rp.nadutils.SaveDeviceInfoFile(resource, id, &devInfo); err != nil {
			return err
		}
	}
	return nil
}

// CleanDeviceInfoFile cleans the Device Info files
func (rp *netResourcePool) CleanDeviceInfoFile(resourceNamePrefix string) error {
	errors := make([]string, 0)
	for id := range rp.GetDevicePool() {
		resource := fmt.Sprintf("%s/%s", resourceNamePrefix, rp.GetConfig().ResourceName)
		if err := rp.nadutils.CleanDeviceInfoFile(resource, id); err != nil {
			// Continue trying to clean.
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ","))
	}
	return nil
}
