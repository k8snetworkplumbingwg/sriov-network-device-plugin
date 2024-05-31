package resources

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

type pKeySelector struct {
	pKeys []string
}

// NewPKeySelector returns a DeviceSelector interface to filter devices based on available PKeys
func NewPKeySelector(pKeys []string) types.DeviceSelector {
	return &pKeySelector{pKeys: pKeys}
}

func (ds *pKeySelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)

	for _, dev := range inDevices {
		pKey := dev.(types.PciNetDevice).GetPKey()
		if pKey != "" && contains(ds.pKeys, pKey) {
			filteredList = append(filteredList, dev)
		}
	}

	return filteredList
}
