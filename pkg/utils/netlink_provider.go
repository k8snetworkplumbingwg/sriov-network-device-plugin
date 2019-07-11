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

var (
	// getLinkByName is a function that retrieves nl.Link object according to
	// a provided netdev name.
	getLinkByName = nl.LinkByName
)

// GetLinkAttrs returns a net device's link attributes.
func GetLinkAttrs(ifName string) (*nl.LinkAttrs, error) {
	link, err := getLinkByName(ifName)
	if err != nil {
		return nil, fmt.Errorf("error getting link attributes for net device %s %v", ifName, err)
	}
	return link.Attrs(), nil
}

type fakeLink struct {
	nl.LinkAttrs
}

func (fl fakeLink) Attrs() *nl.LinkAttrs {
	return &fl.LinkAttrs
}

func (fl fakeLink) Type() string {
	return "fakeType"
}

// GetFakeLinkByName retrieve a fake nl.Link object
func getFakeLinkByName(string) (nl.Link, error) {
	attrs := nl.LinkAttrs{EncapType: "fakeLinkType"}
	return fakeLink{LinkAttrs: attrs}, nil
}

// UseFakeLinks causes GetLinkByName to retrieve fake netlink Link object
// return value intended to be used in unit-tests as deffered method to
// restore GetLinkByName to be the actual netlink implementation.
func UseFakeLinks() func() {
	getLinkByName = getFakeLinkByName
	return func() {
		getLinkByName = nl.LinkByName
	}
}
