// Copyright 2018 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"errors"
	"fmt"

	nl "github.com/vishvananda/netlink"
)

// NetlinkProvider is a wrapper type over netlink library
type NetlinkProvider interface {
	// GetLinkAttrs returns a net device's link attributes.
	GetLinkAttrs(ifName string) (*nl.LinkAttrs, error)
	// GetDevLinkDeviceEswitchAttrs returns a devlink device's attributes
	GetDevLinkDeviceEswitchAttrs(ifName string) (*nl.DevlinkDevEswitchAttr, error)
	// GetIPv4RouteList returns a list of IPv4 routes for specified interface
	GetIPv4RouteList(ifName string) ([]nl.Route, error)
	// DevlinkGetDeviceInfoByNameAsMap returns devlink info for selected device as a map
	GetDevlinkGetDeviceInfoByNameAsMap(bus, device string) (map[string]string, error)
}

type defaultNetlinkProvider struct {
}

var netlinkProvider NetlinkProvider = &defaultNetlinkProvider{}

// Implement the DevlinkGetDeviceInfoByNameAsMap method
func (defaultNetlinkProvider) GetDevlinkGetDeviceInfoByNameAsMap(bus, device string) (map[string]string, error) {
	return nl.DevlinkGetDeviceInfoByNameAsMap(bus, device)
}

// GetNetlinkProvider will be invoked by functions in other packages that would need access to the netlink library
func GetNetlinkProvider() NetlinkProvider {
	return netlinkProvider
}

// GetLinkAttrs returns a net device's link attributes.
func (defaultNetlinkProvider) GetLinkAttrs(ifName string) (*nl.LinkAttrs, error) {
	link, err := nl.LinkByName(ifName)
	if err != nil {
		return nil, fmt.Errorf("error getting link attributes for net device %s %v", ifName, err)
	}
	return link.Attrs(), nil
}

// GetDevLinkDeviceEswitchAttrs returns a devlink device's attributes
func (defaultNetlinkProvider) GetDevLinkDeviceEswitchAttrs(pfAddr string) (*nl.DevlinkDevEswitchAttr, error) {
	dev, err := nl.DevLinkGetDeviceByName("pci", pfAddr)
	if err != nil {
		return nil, fmt.Errorf("error getting devlink device attributes for net device %s %v", pfAddr, err)
	}
	return &(dev.Attrs.Eswitch), nil
}

// GetIPv4RouteList returns a list of IPv4 routes for specified interface
func (defaultNetlinkProvider) GetIPv4RouteList(ifName string) ([]nl.Route, error) {
	link, err := nl.LinkByName(ifName)
	if err != nil {
		return []nl.Route{}, err
	}
	return nl.RouteList(link, nl.FAMILY_V4)
}

// SetNetlinkProviderInst sets a passed instance of NetlinkProvider to be used by unit test in other packages
func SetNetlinkProviderInst(inst NetlinkProvider) {
	netlinkProvider = inst
}

const (
	// fwAppNameKey is a key to extract DDP name
	fwAppNameKey = "fw.app.name"
	pciBus       = "pci"
)

// IsDevlinkDDPSupportedByDevice checks if DDP of device can be acquired with devlink info command
func IsDevlinkDDPSupportedByDevice(device string) bool {
	if _, err := netlinkProvider.GetDevlinkGetDeviceInfoByNameAsMap(device, fwAppNameKey); err != nil {
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
	data, err := netlinkProvider.GetDevlinkGetDeviceInfoByNameAsMap(pciBus, device)
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
