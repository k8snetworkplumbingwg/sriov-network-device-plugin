package utils

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

// NetlinkFunction is a type of function used by netlink to get data
type NetlinkFunction func(string, string) (map[string]string, error)

const (
	// fwAppNameKey is a key to extract DDP name
	fwAppNameKey = "fw.app.name"
	pciBus       = "pci"
)

var (
	netlinkDevlinkGetDeviceInfoByNameAsMap = netlink.DevlinkGetDeviceInfoByNameAsMap
)

// IsDevlinkDDPSupportedByDevice checks if DDP of device can be acquired with devlink info command
func IsDevlinkDDPSupportedByDevice(device string) bool {
	if _, err := devlinkGetDeviceInfoByNameAndKey(device, fwAppNameKey); err != nil {
		return false
	}

	return true
}

// DevlinkGetDDPProfiles returns DDP for selected device
func DevlinkGetDDPProfiles(device string) (string, error) {
	return devlinkGetDeviceInfoByNameAndKey(device, fwAppNameKey)
}

// DevlinkGetDeviceInfoByNameAndKeys returns values for selected keys in device info
func DevlinkGetDeviceInfoByNameAndKeys(device string, keys []string) (map[string]string, error) {
	data, err := netlinkDevlinkGetDeviceInfoByNameAsMap(pciBus, device)
	if err != nil {
		return nil, err
	}

	info := make(map[string]string)

	for _, key := range keys {
		if value, exists := data[key]; exists {
			info[key] = value
		} else {
			return nil, keyNotFoundError("DevlinkGetDeviceInfoByNameAndKeys", key)
		}
	}

	return info, nil
}

// DevlinkGetDeviceInfoByNameAndKey returns values for selected key in device infol
func devlinkGetDeviceInfoByNameAndKey(device, key string) (string, error) {
	keys := []string{key}
	info, err := DevlinkGetDeviceInfoByNameAndKeys(device, keys)
	if err != nil {
		return "", err
	}

	return info[key], nil
}

// ErrKeyNotFound error when key is not found in the parsed response
var ErrKeyNotFound = errors.New("key could not be found")

// KeyNotFoundError returns ErrKeyNotFound
func keyNotFoundError(function, key string) error {
	return fmt.Errorf("%s - %w: %s", function, ErrKeyNotFound, key)
}
