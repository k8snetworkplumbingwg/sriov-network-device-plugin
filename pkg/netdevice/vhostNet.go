package netdevice

import (
	"os"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// VhostNetDeviceExist returns true if /dev/vhost-net exists
func VhostNetDeviceExist() bool {
	_, err := os.Stat("/dev/vhost-net")
	return err == nil
}

// GetVhostNetDeviceSpec returns an instance of DeviceSpec for vhost-net
func GetVhostNetDeviceSpec() []*pluginapi.DeviceSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      "/dev/vhost-net",
		ContainerPath: "/dev/vhost-net",
		Permissions:   "mrw",
	})

	return deviceSpec
}

// TunDeviceExist returns true if /dev/net/tun exists
func TunDeviceExist() bool {
	_, err := os.Stat("/dev/net/tun")
	return err == nil
}

// GetTunDeviceSpec returns an instance of DeviceSpec for Tun
func GetTunDeviceSpec() []*pluginapi.DeviceSpec {
	deviceSpec := make([]*pluginapi.DeviceSpec, 0)
	deviceSpec = append(deviceSpec, &pluginapi.DeviceSpec{
		HostPath:      "/dev/net/tun",
		ContainerPath: "/dev/net/tun",
		Permissions:   "mrw",
	})

	return deviceSpec
}
