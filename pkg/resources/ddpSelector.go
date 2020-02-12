package resources

import (
	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

type ddpSelector struct {
	profiles []string
}

// newDdpSelector returns a DeviceSelector interface to filter devices based on available DDP profile
func newDdpSelector(profiles []string) types.DeviceSelector {
	return &ddpSelector{profiles: profiles}
}

func (ds *ddpSelector) Filter(inDevices []types.PciNetDevice) []types.PciNetDevice {
	filteredList := make([]types.PciNetDevice, 0)

	for _, dev := range inDevices {
		ddpProfile := dev.GetDDPProfiles()
		if ddpProfile != "" && contains(ds.profiles, ddpProfile) {
			filteredList = append(filteredList, dev)
		}
	}

	return filteredList
}
