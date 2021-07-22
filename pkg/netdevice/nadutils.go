package netdevice

import (
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	nadutils "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/utils"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

// nadutils implements types.NadUtils interface
// It's purpose is to wrap the utilities provided by github.com/k8snetworkplumbingwg/network-attachment-definition-client
// in order to make mocking easy for Unit Tests
type nadUtils struct {
}

func (nu *nadUtils) SaveDeviceInfoFile(resourceName, deviceID string, devInfo *nettypes.DeviceInfo) error {
	return nadutils.SaveDeviceInfoForDP(resourceName, deviceID, devInfo)
}

func (nu *nadUtils) CleanDeviceInfoFile(resourceName, deviceID string) error {
	return nadutils.CleanDeviceInfoForDP(resourceName, deviceID)
}

// NewNadUtils returns a new NadUtils
func NewNadUtils() types.NadUtils {
	return &nadUtils{}
}
