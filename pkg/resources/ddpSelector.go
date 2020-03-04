package resources

import (
	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

type ddpSelector struct {
	profiles []string
}

// NewDdpSelector returns a DeviceSelector interface to filter devices based on available DDP profile
func NewDdpSelector(profiles []string) types.DeviceSelector {
	return &ddpSelector{profiles: profiles}
}

func (ds *ddpSelector) Filter(inDevices []types.PciDevice) []types.PciDevice {
	filteredList := make([]types.PciDevice, 0)

	for _, dev := range inDevices {
		ddpProfile := dev.(types.PciNetDevice).GetDDPProfiles()
		if ddpProfile != "" && contains(ds.profiles, ddpProfile) {
			filteredList = append(filteredList, dev)
		}
	}

	return filteredList
}
