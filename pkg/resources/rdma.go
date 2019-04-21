package resources

import (
	"github.com/Mellanox/rdmamap"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

type rdmaSpec struct {
	isSupportRdma bool
	deviceSpec    []*pluginapi.DeviceSpec
}

// NewRdmaSpec returns the RdmaSpec
func NewRdmaSpec(pciAddrs string) types.RdmaSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	isSupportRdma := false
	rdmaResources := rdmamap.GetRdmaDevicesForPcidev(pciAddrs)
	if len(rdmaResources) > 0 {
		isSupportRdma = true
		for _, res := range rdmaResources {
			resRdmaDevices := rdmamap.GetRdmaCharDevices(res)
			for _, rdmaDevice := range resRdmaDevices {
				deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
					HostPath:      rdmaDevice,
					ContainerPath: rdmaDevice,
					Permissions:   "rwm",
				})
			}
		}
	}

	return &rdmaSpec{isSupportRdma: isSupportRdma, deviceSpec: deviceSpec}
}

func (r *rdmaSpec) IsRdma() bool {
	return r.isSupportRdma
}

func (r *rdmaSpec) GetRdmaDeviceSpec() []*pluginapi.DeviceSpec {
	return r.deviceSpec
}
