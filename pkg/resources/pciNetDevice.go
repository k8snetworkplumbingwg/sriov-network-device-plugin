package resources

import (
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	"github.com/jaypipes/ghw"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type pciNetDevice struct {
	pciDevice   *ghw.PCIDevice
	ifName      string
	pfName      string
	pfAddr      string
	driver      string
	vendor      string
	product     string
	vfID        int
	linkSpeed   string
	env         string
	numa        string
	apiDevice   *pluginapi.Device
	deviceSpecs []*pluginapi.DeviceSpec
	mounts      []*pluginapi.Mount
}

// NewPciNetDevice returns an instance of PciNetDevice interface
func NewPciNetDevice(pciDevice *ghw.PCIDevice, rFactory types.ResourceFactory) (types.PciNetDevice, error) {
	// populate all fields in pciNetDevice here

	// 			1. get PF details, add PF info in its pciNetDevice struct
	// 			2. Get driver info
	pciAddr := pciDevice.Address
	driverName, err := utils.GetDriverName(pciAddr)
	if err != nil {
		return nil, err
	}

	// 			3. Get Device file info (e.g., uio, vfio specific)
	// Get DeviceInfoProvider using device driver
	infoProvider := rFactory.GetInfoProvider(driverName)
	dSpecs := infoProvider.GetDeviceSpecs(pciAddr)
	mnt := infoProvider.GetMounts(pciAddr)
	env := infoProvider.GetEnvVal(pciAddr)
	apiDevice := &pluginapi.Device{
		ID:     pciAddr,
		Health: pluginapi.Healthy,
	}

	// 			4. Create pciNetDevice object with all relevent info
	return &pciNetDevice{
		pciDevice:   pciDevice,
		ifName:      "", // TO-DO: Get this using utils pkg
		pfName:      "", // TO-DO: Get this using utils pkg
		driver:      driverName,
		vfID:        0,  // TO-DO: Get this using utils pkg if needed
		linkSpeed:   "", // TO-DO: Get this using utils pkg
		apiDevice:   apiDevice,
		deviceSpecs: dSpecs,
		mounts:      mnt,
		env:         env,
		numa:        "", // TO-DO: Get this using utils pkg
	}, nil
}

func (nd *pciNetDevice) GetPFName() string {
	return nd.pfName
}

func (nd *pciNetDevice) GetPfPciAddr() string {
	return nd.pfAddr
}

func (nd *pciNetDevice) GetVendor() string {
	return nd.pciDevice.Vendor.ID
}

func (nd *pciNetDevice) GetDeviceCode() string {
	return nd.pciDevice.Product.ID
}
func (nd *pciNetDevice) GetPciAddr() string {
	return nd.pciDevice.Address
}
func (nd *pciNetDevice) GetDriver() string {
	return nd.driver
}
func (nd *pciNetDevice) IsSriovPF() bool {
	return false
}
func (nd *pciNetDevice) GetLinkSpeed() string {
	return nd.linkSpeed
}
func (nd *pciNetDevice) GetSubClass() string {
	return nd.pciDevice.Subclass.ID
}

func (nd *pciNetDevice) GetDeviceSpecs() []*pluginapi.DeviceSpec {
	return nd.deviceSpecs
}

func (nd *pciNetDevice) GetEnvVal() string {
	return nd.env
}

func (nd *pciNetDevice) GetMounts() []*pluginapi.Mount {
	return nd.mounts
}

func (nd *pciNetDevice) GetAPIDevice() *pluginapi.Device {
	return nd.apiDevice
}

func getPFInfos(pciAddr string) (pfAddr, pfName string, err error) {
	return "", "", nil
}
