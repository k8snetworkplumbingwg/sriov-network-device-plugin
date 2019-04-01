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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/golang/glog"
	"github.com/jaypipes/ghw"
	"github.com/vishvananda/netlink"

	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
)

const (
	socketSuffix = "sock"
	netClass     = 0x02 // Device class - Network controller.	 ref: https://pci-ids.ucw.cz/read/PD/02 (for Sub-Classes)
)

/*
Network controller subclasses. ref: https://pci-ids.ucw.cz/read/PD/02
		00	Ethernet controller
		01	Token ring network controller
		02	FDDI network controller
		03	ATM network controller
		04	ISDN controller
		05	WorldFip controller
		06	PICMG controller
		07	Infiniband controller
		08	Fabric controller
		80	Network controller
*/

type cliParams struct {
	configFile     string
	resourcePrefix string
}

type resourceManager struct {
	cliParams
	pluginWatchMode bool
	socketSuffix    string
	rFactory        types.ResourceFactory
	configList      []*types.ResourceConfig // resourceName -> resourcePool
	resourceServers []types.ResourceServer
	netDeviceList   []types.PciNetDevice         // all network devices in host
	linkWatchList   map[string]types.LinkWatcher // SRIOV PF list - for watching link status
}

func newResourceManager(cp *cliParams) *resourceManager {
	pluginWatchMode := utils.DetectPluginWatchMode(types.SockDir)
	if pluginWatchMode {
		glog.Infof("Using Kubelet Plugin Registry Mode")
	} else {
		glog.Infof("Using Deprecated Devie Plugin Registry Path")
	}
	return &resourceManager{
		cliParams:       *cp,
		pluginWatchMode: pluginWatchMode,
		rFactory:        resources.NewResourceFactory(cp.resourcePrefix, socketSuffix, pluginWatchMode),
		netDeviceList:   make([]types.PciNetDevice, 0),
		linkWatchList:   make(map[string]types.LinkWatcher, 0),
	}
}

// Read and validate configurations from Config file
func (rm *resourceManager) readConfig() error {

	resources := &types.ResourceConfList{}
	rawBytes, err := ioutil.ReadFile(rm.configFile)

	if err != nil {
		return fmt.Errorf("error reading file %s, %v", rm.configFile, err)

	}

	if err = json.Unmarshal(rawBytes, resources); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v", err)
	}

	glog.Infof("ResourceList: %+v", resources.ResourceList)
	for i := range resources.ResourceList {
		rm.configList = append(rm.configList, &resources.ResourceList[i])
	}

	return nil
}

func (rm *resourceManager) initServers() error {
	rf := rm.rFactory
	glog.Infof("number of config: %d\n", len(rm.configList))
	for _, rc := range rm.configList {
		// Create new ResourcePool
		glog.Infof("")
		glog.Infof("Creating new ResourcePool: %s", rc.ResourceName)
		rPool, err := rm.rFactory.GetResourcePool(rc, rm.netDeviceList)
		if err != nil {
			glog.Errorf("initServers(): error creating ResourcePool with config %+v: %q", rc, err)
			return err
		}

		// Create ResourceServer with this ResourcePool
		s, err := rf.GetResourceServer(rPool)
		if err != nil {
			glog.Errorf("initServers(): error creating ResourceServer: %v", err)
			return err
		}
		glog.Infof("New resource server is created for %s ResourcePool", rc.ResourceName)
		rm.resourceServers = append(rm.resourceServers, s)
	}
	return nil
}

func (rm *resourceManager) startAllServers() error {
	for _, rs := range rm.resourceServers {
		if err := rs.Start(); err != nil {
			return err
		}

		// start watcher
		if !rm.pluginWatchMode {
			go rs.Watch()
		}
	}
	return nil
}

func (rm *resourceManager) stopAllServers() error {
	for _, rs := range rm.resourceServers {
		if err := rs.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// Validate configurations
func (rm *resourceManager) validConfigs() bool {
	resourceName := make(map[string]string) // resource name placeholder

	for _, conf := range rm.configList {
		// check if name contains acceptable characters
		if !utils.ValidResourceName(conf.ResourceName) {
			glog.Errorf("resource name \"%s\" contains invalid characters", conf.ResourceName)
			return false
		}
		// check resource names are unique
		_, ok := resourceName[conf.ResourceName]
		if ok {
			// resource name already exist
			glog.Errorf("resource name \"%s\" already exists", conf.ResourceName)
			return false
		}

		resourceName[conf.ResourceName] = conf.ResourceName
	}

	return true
}

func (rm *resourceManager) discoverHostDevices() error {
	glog.Infoln("discovering host network devices")

	pci, err := ghw.PCI()
	if err != nil {
		return fmt.Errorf("discoverDevices(): error getting PCI info: %v", err)
	}

	devices := pci.ListDevices()
	if len(devices) == 0 {
		glog.Warningf("discoverDevices(): no PCI network device found")
	}
	for _, device := range devices {
		devClass, err := strconv.ParseInt(device.Class.ID, 16, 64)
		if err != nil {
			glog.Warningf("discoverDevices(): unable to parse device class for device %+v %q", device, err)
			continue
		}

		// only interested in network class
		if devClass == netClass {
			vendor := device.Vendor
			vendorName := vendor.Name
			if len(vendor.Name) > 20 {
				vendorName = string([]byte(vendorName)[0:17]) + "..."
			}
			product := device.Product
			productName := product.Name
			if len(product.Name) > 40 {
				productName = string([]byte(productName)[0:37]) + "..."
			}
			glog.Infof("discoverDevices(): device found: %-12s\t%-12s\t%-20s\t%-40s", device.Address, device.Class.ID, vendorName, productName)

			// exclude device in-use in host
			if isUsed, _ := isInUse(device.Address); !isUsed {

				aPF := utils.IsSriovPF(device.Address)
				aVF := utils.IsSriovVF(device.Address)

				if aPF || !aVF {
					// add to linkWatchList
					rm.addToLinkWatchList(device.Address)
				}

				if aPF && utils.SriovConfigured(device.Address) {
					// do not add this device in net device list
					continue
				}

				if newDevice, err := resources.NewPciNetDevice(device, rm.rFactory); err == nil {
					rm.netDeviceList = append(rm.netDeviceList, newDevice)
				} else {
					glog.Errorf("discoverDevices() error adding new device: %q", err)
				}

			}
		}
	}
	return nil
}

// isInUse returns true if PCI network device appear to be in use by host system given its pci address as string.
// Otherwise it is assumed not to be in used.
func isInUse(pciAddr string) (bool, error) {

	// inUse := false
	// Get net interface name
	ifNames, err := utils.GetNetNames(pciAddr)
	if err != nil {
		return false, fmt.Errorf("error trying get net device name for device %s", pciAddr)
	}

	if len(ifNames) > 0 { // there's at least one interface name found
		for _, ifName := range ifNames {
			link, err := netlink.LinkByName(ifName)
			if err != nil {
				glog.Errorf("expected to get valid host interface with name %s: %q", ifName, err)
			}

			routes, err := netlink.RouteList(link, netlink.FAMILY_V4) // IPv6 routes: all interface has at least one link local route entry
			if len(routes) > 0 {
				glog.Infof("excluding interface %s:  route entry found: %+v", ifName, routes)
				return true, nil
			}
		}
	}

	return false, nil
}

func (rm *resourceManager) addToLinkWatchList(pciAddr string) {
	netNames, err := utils.GetNetNames(pciAddr)
	if err == nil {
		// There are some cases, where we may get multiple netdevice name for a PCI device
		// Only add one device
		if len(netNames) > 0 {
			netName := netNames[0]
			lw := &linkWatcher{ifName: netName}
			if _, ok := rm.linkWatchList[pciAddr]; !ok {
				rm.linkWatchList[netName] = lw
				glog.Infof("%s added to linkWatchList", netName)
			}
		}
	}
}

type linkWatcher struct {
	ifName string
	// subscribers []LinkSubscriber
}

func (lw *linkWatcher) Subscribe() {

}
