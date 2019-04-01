package resources

import (
	"github.com/Mellanox/rdmamap"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

const MLNX_RDMA_DIR = "/dev/infiniband"

type mlnxRdmaSpec struct {
	isRdma bool
	mounts []*pluginapi.Mount
}

func newMlnxRdmaSpec(pciAddrs string) types.RdmaSpec {
	isRdma := len(rdmamap.GetRdmaDevicesForPcidev(pciAddrs)) > 0
	mounts := make([]*pluginapi.Mount, 0)
	mounts = append(mounts, &pluginapi.Mount{
		HostPath:      MLNX_RDMA_DIR,
		ContainerPath: MLNX_RDMA_DIR,
		ReadOnly:      false,
	})
	return &mlnxRdmaSpec{isRdma: isRdma, mounts: mounts}
}

func (r *mlnxRdmaSpec) IsRdma() bool {
	return r.isRdma
}

func (r *mlnxRdmaSpec) GetMounts() []*pluginapi.Mount {
	return r.mounts
}
