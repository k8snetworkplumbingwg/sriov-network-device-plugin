package checkpoint

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/checkpointmanager/checksum"
)

const (
	checkPointfile                 = "/var/lib/kubelet/device-plugins/kubelet_internal_checkpoint"
	pluginDir                      = pluginapi.DevicePluginPath
	kubeletDeviceManagerCheckpoint = "kubelet_internal_checkpoint"
)

type PodDevicesEntry struct {
	PodUID        string
	ContainerName string
	ResourceName  string
	DeviceIDs     []string
	AllocResp     []byte
}

type checkpointData struct {
	PodDeviceEntries  []PodDevicesEntry
	RegisteredDevices map[string][]string
}

type Data struct {
	Data     checkpointData
	Checksum checksum.Checksum
}

// Get all Pod entires for given resourceName
func GetPodEntries(resourceName string) ([]PodDevicesEntry, error) {

	podEntries := []PodDevicesEntry{}

	cpd := &Data{}
	rawBytes, err := ioutil.ReadFile(checkPointfile)
	if err != nil {
		return podEntries, fmt.Errorf("Error reading file %s\n%v\n", checkPointfile, err)

	}

	if err = json.Unmarshal(rawBytes, cpd); err != nil {
		return podEntries, fmt.Errorf("Error unmarshalling raw bytes %v", err)
	}

	responseData := cpd.Data.PodDeviceEntries

	for _, p := range responseData {
		if p.ResourceName == resourceName {
			podEntries = append(podEntries, p)
		}
	}
	return podEntries, nil
}
