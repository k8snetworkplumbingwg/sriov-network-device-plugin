package rdma

import (
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

type genericRdmaSpec struct {
	isRdma     bool
	deviceSpec []*pluginapi.DeviceSpec
}

// NewGenericRdmaSpec returns the Mellanox vendor RdmaSpec
func NewGenericRdmaSpec() types.RdmaSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	return &genericRdmaSpec{isRdma: false, deviceSpec: deviceSpec}
}

func (r *genericRdmaSpec) IsRdma() bool {
	return r.isRdma
}

func (r *genericRdmaSpec) GetDeviceSpec() []*pluginapi.DeviceSpec {
	return r.deviceSpec
}
