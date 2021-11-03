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
	"fmt"

	nl "github.com/vishvananda/netlink"
)

// NetlinkProvider is a wrapper type over netlink library
type NetlinkProvider interface {
	// GetLinkAttrs returns a net device's link attributes.
	GetLinkAttrs(ifName string) (*nl.LinkAttrs, error)
	// GetDevLinkDeviceEswitchAttrs returns a devlink device's attributes
	GetDevLinkDeviceEswitchAttrs(ifName string) (*nl.DevlinkDevEswitchAttr, error)
}

type defaultNetlinkProvider struct {
}

var netlinkProvider NetlinkProvider = &defaultNetlinkProvider{}

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

// SetNetlinkProviderInst sets a passed instance of NetlinkProvider to be used by unit test in other packages
func SetNetlinkProviderInst(inst NetlinkProvider) {
	netlinkProvider = inst
}
