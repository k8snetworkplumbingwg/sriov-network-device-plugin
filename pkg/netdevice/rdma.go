package netdevice

import (
	"io/ioutil"
	"path/filepath"

	"github.com/Mellanox/rdmamap"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

type rdmaSpec struct {
	isSupportRdma bool
	deviceSpec    []*pluginapi.DeviceSpec
}

func newRdmaSpec(rdmaResources []string) types.RdmaSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	isSupportRdma := false
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

// NewRdmaSpec returns the RdmaSpec for pci device
func NewRdmaSpec(pciAddrs string) types.RdmaSpec {
	rdmaResources := rdmamap.GetRdmaDevicesForPcidev(pciAddrs)
	return newRdmaSpec(rdmaResources)
}

func getRdmaDevicesForAuxdev(auxDev string) []string {
	var rdmaResources []string

	dirName := filepath.Join(utils.SysBusAux, auxDev, rdmamap.RdmaClassName)

	entries, err := ioutil.ReadDir(dirName)
	if err != nil {
		return rdmaResources
	}

	for _, entry := range entries {
		if entry.IsDir() == false {
			continue
		}
		rdmaResources = append(rdmaResources, entry.Name())
	}
	return rdmaResources
}

// NewAuxRdmaSpec returns the RdmaSpec for auxiliary device
func NewAuxRdmaSpec(auxID string) types.RdmaSpec {
	rdmaResources := getRdmaDevicesForAuxdev(auxID)
	return newRdmaSpec(rdmaResources)
}

func (r *rdmaSpec) IsRdma() bool {
	return r.isSupportRdma
}

func (r *rdmaSpec) GetRdmaDeviceSpec() []*pluginapi.DeviceSpec {
	return r.deviceSpec
}
