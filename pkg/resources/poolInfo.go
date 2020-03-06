package resources

import (
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	"strconv"
)

// PoolInfoImpl implements PoolInfo interface and can serve as a base class for
// more advanced device-specific PoolInfo instances
type PoolInfoImpl struct {
	PciDev      types.PciDevice
	Env         string
	APIDevice   *pluginapi.Device
	DeviceSpecs []*pluginapi.DeviceSpec
	Mounts      []*pluginapi.Mount
}

// NewPoolInfoImpl creates a new PoolInfoImpl with a given DeviceInfoProvider
func NewPoolInfoImpl(pciDev types.PciDevice, rc *types.ResourceConfig, ip types.DeviceInfoProvider) (*PoolInfoImpl, error) {
	env := ip.GetEnvVal(pciDev.GetPciAddr())
	mnt := ip.GetMounts(pciDev.GetPciAddr())

	// Set DeviceSpecs
	dSpecs := ip.GetDeviceSpecs(pciDev.GetPciAddr())
	// Create apiDevice
	apiDevice := &pluginapi.Device{
		ID:     pciDev.GetPciAddr(),
		Health: pluginapi.Healthy,
	}
	nodeNum := utils.GetDevNode(pciDev.GetPciAddr())
	if nodeNum >= 0 {
		numaInfo := &pluginapi.NUMANode{
			ID: int64(nodeNum),
		}
		apiDevice.Topology = &pluginapi.TopologyInfo{
			Nodes: []*pluginapi.NUMANode{numaInfo},
		}
	}
	return &PoolInfoImpl{
		PciDev:      pciDev,
		Env:         env,
		APIDevice:   apiDevice,
		DeviceSpecs: dSpecs,
		Mounts:      mnt,
	}, nil
}

// Convert NUMA node number to string.
// A node of -1 represents "unknown" and is converted to the empty string.
func nodeToStr(nodeNum int) string {
	if nodeNum >= 0 {
		return strconv.Itoa(nodeNum)
	}
	return ""
}

// GetDeviceSpecs returns the Device Specifications
func (pd *PoolInfoImpl) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	return pd.DeviceSpecs
}

// GetEnvVal returns the ENV variable value
func (pd *PoolInfoImpl) GetEnvVal() string {
	return pd.Env
}

// GetMounts returns the Mount list
func (pd *PoolInfoImpl) GetMounts() []*pluginapi.Mount {
	return pd.Mounts
}

// GetAPIDevice returns the API Device
func (pd *PoolInfoImpl) GetAPIDevice() *pluginapi.Device {
	return pd.APIDevice
}
