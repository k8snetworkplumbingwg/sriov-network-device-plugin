package rdma

import (
	"github.com/Mellanox/rdmamap"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

// mlnxRdmaDir path to rdma resources
const mlnxRdmaDir = "/dev/infiniband"

type mlnxRdmaSpec struct {
	isRdma     bool
	deviceSpec []*pluginapi.DeviceSpec
}

// NewMlnxRdmaSpec returns the Mellanox vendor RdmaSpec
func NewMlnxRdmaSpec(pciAddrs string) types.RdmaSpec {
	isRdma := len(rdmamap.GetRdmaDevicesForPcidev(pciAddrs)) > 0
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      mlnxRdmaDir,
		ContainerPath: mlnxRdmaDir,
		Permissions:   "rwm",
	})
	return &mlnxRdmaSpec{isRdma: isRdma, deviceSpec: deviceSpec}
}

func (r *mlnxRdmaSpec) IsRdma() bool {
	return r.isRdma
}

func (r *mlnxRdmaSpec) GetDeviceSpec() []*pluginapi.DeviceSpec {
	return r.deviceSpec
}
