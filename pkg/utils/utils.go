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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

var (
	sysBusPci = "/sys/bus/pci/devices"
	// golangci-lint doesn't see it is used in the testing.go
	//nolint: unused
	sysBusAux = "/sys/bus/auxiliary/devices"
	devDir    = "/dev"
)

const (
	totalVfFile          = "sriov_totalvfs"
	configuredVfFile     = "sriov_numvfs"
	eswitchModeSwitchdev = "switchdev"
	classIDBaseInt       = 16
	classIDBitSize       = 64
	maxVendorName        = 20
	maxProductName       = 40
)

// DetectPluginWatchMode returns true if plugins registry directory exist
func DetectPluginWatchMode(sockDir string) bool {
	if _, err := os.Stat(sockDir); err != nil {
		return false
	}
	return true
}

// GetPfAddr returns SRIOV PF pci address if a device is VF given its pci address.
// If device it not VF then it will return empty string
func GetPfAddr(pciAddr string) (string, error) {
	pfSymLink := filepath.Join(sysBusPci, pciAddr, "physfn")
	pciinfo, err := os.Readlink(pfSymLink)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("error getting PF for PCI device %s %v", pciAddr, err)
	}
	return filepath.Base(pciinfo), nil
}

// GetPfName returns SRIOV PF name for the given VF
// If device is not VF then it will return empty string
func GetPfName(pciAddr string) (string, error) {
	if !IsSriovVF(pciAddr) {
		return "", nil
	}

	pfEswitchMode, err := GetPfEswitchMode(pciAddr)
	if pfEswitchMode == "" {
		// If device doesn't support eswitch mode query or doesn't have sriov enabled,
		// fall back to the default implementation
		if err == nil || strings.Contains(strings.ToLower(err.Error()), "error getting devlink device attributes for net device") {
			glog.Infof("Devlink query for eswitch mode is not supported for device %s. %v", pciAddr, err)
		} else {
			return "", err
		}
	} else if pfEswitchMode == eswitchModeSwitchdev {
		name, err := GetSriovnetProvider().GetUplinkRepresentor(pciAddr)
		if err != nil {
			return "", err
		}

		return name, nil
	}

	path := filepath.Join(sysBusPci, pciAddr, "physfn", "net")
	files, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	} else if len(files) > 0 {
		return files[0].Name(), nil
	}
	return "", fmt.Errorf("the PF name is not found for device %s", pciAddr)
}

// IsSriovPF check if a pci device SRIOV capable given its pci address
func IsSriovPF(pciAddr string) bool {
	totalVfFilePath := filepath.Join(sysBusPci, pciAddr, totalVfFile)
	if _, err := os.Stat(totalVfFilePath); err != nil {
		return false
	}
	// sriov_totalvfs file exists -> sriov capable
	return true
}

// IsSriovVF check if a pci device has link to a PF
func IsSriovVF(pciAddr string) bool {
	totalVfFilePath := filepath.Join(sysBusPci, pciAddr, "physfn")
	if _, err := os.Stat(totalVfFilePath); err != nil {
		return false
	}
	return true
}

// GetVFconfigured returns number of VF configured for a PF
func GetVFconfigured(pf string) int {
	configuredVfPath := filepath.Join(sysBusPci, pf, configuredVfFile)
	vfs, err := os.ReadFile(configuredVfPath)
	if err != nil {
		return 0
	}
	configuredVFs := bytes.TrimSpace(vfs)
	numConfiguredVFs, err := strconv.Atoi(string(configuredVFs))
	if err != nil {
		return 0
	}
	return numConfiguredVFs
}

// GetVFList returns a List containing PCI addr for all VF discovered in a given PF
func GetVFList(pf string) (vfList []string, err error) {
	vfList = make([]string, 0)
	pfDir := filepath.Join(sysBusPci, pf)
	_, err = os.Lstat(pfDir)
	if err != nil {
		err = fmt.Errorf("error. Could not get PF directory information for device: %s, Err: %v", pf, err)
		return
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))

	if err != nil {
		err = fmt.Errorf("error reading VF directories %v", err)
		return
	}

	// Read all VF directory and get add VF PCI addr to the vfList
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				vfLink := filepath.Base(linkName)
				vfList = append(vfList, vfLink)
			}
		}
	}
	return
}

// GetPciAddrFromVFID returns PCI address for VF ID
func GetPciAddrFromVFID(pf string, vf int) (pciAddr string, err error) {
	vfDir := fmt.Sprintf("%s/%s/virtfn%d", sysBusPci, pf, vf)
	dirInfo, err := os.Lstat(vfDir)
	if err != nil {
		err = fmt.Errorf("could not get directory information for device: %s, VF: %v. Err: %v", pf, vf, err)
		return "", err
	}

	if (dirInfo.Mode() & os.ModeSymlink) == 0 {
		err = fmt.Errorf("no symbolic link between virtual function and PCI - Device: %s, VF: %v", pf, vf)
		return
	}

	pciInfo, err := os.Readlink(vfDir)
	if err != nil {
		err = fmt.Errorf("cannot read symbolic link between virtual function and PCI - Device: %s, VF: %v. Err: %v", pf, vf, err)
		return
	}

	pciAddr = pciInfo[len("../"):]
	return
}

// GetSriovVFcapacity returns SRIOV VF capacity
func GetSriovVFcapacity(pf string) int {
	totalVfFilePath := filepath.Join(sysBusPci, pf, totalVfFile)
	vfs, err := os.ReadFile(totalVfFilePath)
	if err != nil {
		return 0
	}
	totalvfs := bytes.TrimSpace(vfs)
	numvfs, err := strconv.Atoi(string(totalvfs))
	if err != nil {
		return 0
	}
	return numvfs
}

// GetDevNode returns the numa node of a PCI device, -1 if none is specified or error.
func GetDevNode(pciAddr string) int {
	devNodePath := filepath.Join(sysBusPci, pciAddr, "numa_node")
	node, err := os.ReadFile(devNodePath)
	if err != nil {
		return -1
	}
	node = bytes.TrimSpace(node)
	numNode, err := strconv.Atoi(string(node))
	if err != nil {
		return -1
	}
	return numNode
}

// IsNetlinkStatusUp returns 'false' if 'operstate' is not "up" for a Linux network device.
// This function will only return 'false' if the 'operstate' file of the device is readable
// and holds value anything other than "up". Or else we assume link is up.
func IsNetlinkStatusUp(dev string) bool {
	if opsFiles, err := filepath.Glob(filepath.Join(sysBusPci, dev, "net", "*", "operstate")); err == nil {
		for _, f := range opsFiles {
			b, err := os.ReadFile(f)
			if err != nil || strings.TrimSpace(string(b)) != "up" {
				return false
			}
		}
	}
	return true
}

// ValidPciAddr validates PciAddr given as string with host system
func ValidPciAddr(addr string) (string, error) {
	// Check system pci address

	// sysbus pci address regex
	var validLongID = regexp.MustCompile(`^0{4}:[0-9a-f]{2}:[0-9a-f]{2}.[0-7]$`)
	var validShortID = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}.[0-7]$`)

	if validLongID.MatchString(addr) {
		return addr, deviceExist(addr)
	} else if validShortID.MatchString(addr) {
		addr = "0000:" + addr // make short form to sysfs represtation
		return addr, deviceExist(addr)
	}

	return "", fmt.Errorf("invalid pci address %s", addr)
}

func deviceExist(addr string) error {
	devPath := filepath.Join(sysBusPci, addr)
	_, err := os.Lstat(devPath)
	if err != nil {
		return fmt.Errorf("error: unable to read device directory %s", devPath)
	}
	return nil
}

// SriovConfigured returns true if sriov_numvfs reads > 0 else false
func SriovConfigured(addr string) bool {
	return GetVFconfigured(addr) > 0
}

// ValidResourceName returns true if it contains permitted characters otherwise false
func ValidResourceName(name string) bool {
	// name regex
	var validString = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validString.MatchString(name)
}

// GetVFIODeviceFile returns a vfio device files for vfio-pci bound PCI device's PCI address
func GetVFIODeviceFile(dev string) (devFileHost, devFileContainer string, err error) {
	// Get iommu group for this device
	devPath := filepath.Join(sysBusPci, dev)
	_, err = os.Lstat(devPath)
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): Could not get directory information for device: %s, Err: %v", dev, err)
		return devFileHost, devFileContainer, err
	}

	iommuDir := filepath.Join(devPath, "iommu_group")
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): error reading iommuDir %v", err)
		return devFileHost, devFileContainer, err
	}

	dirInfo, err := os.Lstat(iommuDir)
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): unable to find iommu_group %v", err)
		return devFileHost, devFileContainer, err
	}

	if dirInfo.Mode()&os.ModeSymlink == 0 {
		err = fmt.Errorf("GetVFIODeviceFile(): invalid symlink to iommu_group %v", err)
		return devFileHost, devFileContainer, err
	}

	linkName, err := filepath.EvalSymlinks(iommuDir)
	if err != nil {
		err = fmt.Errorf("GetVFIODeviceFile(): error reading symlink to iommu_group %v", err)
		return devFileHost, devFileContainer, err
	}
	devFileContainer = filepath.Join(devDir, "vfio", filepath.Base(linkName))
	devFileHost = devFileContainer

	// Get a file path to the iommu group name
	namePath := filepath.Join(linkName, "name")
	// Read the iommu group name
	// The name file will not exist on baremetal
	vfioName, errName := os.ReadFile(namePath)
	if errName == nil {
		vName := strings.TrimSpace(string(vfioName))

		// if the iommu group name == vfio-noiommu then we are in a VM, adjust path to vfio device
		if vName == "vfio-noiommu" {
			linkName = filepath.Join(filepath.Dir(linkName), "noiommu-"+filepath.Base(linkName))
			devFileHost = filepath.Join(devDir, "vfio", filepath.Base(linkName))
		}
	}

	return devFileHost, devFileContainer, err
}

// GetUIODeviceFile returns a vfio device files for vfio-pci bound PCI device's PCI address
func GetUIODeviceFile(dev string) (devFile string, err error) {
	vfDir := filepath.Join(sysBusPci, dev, "uio")

	_, err = os.Lstat(vfDir)
	if err != nil {
		return "", fmt.Errorf("GetUIODeviceFile(): could not get directory information for device: %s Err: %v", vfDir, err)
	}

	files, err := os.ReadDir(vfDir)

	if err != nil {
		return
	}

	// uio directory should only contain one directory e.g uio1
	// assuption is there's a corresponding device file in /dev e.g. /dev/uio1
	devFile = filepath.Join(devDir, files[0].Name())

	return
}

// GetNetNames returns host net interface names as string for a PCI device from its pci address
func GetNetNames(pciAddr string) ([]string, error) {
	netDir := filepath.Join(sysBusPci, pciAddr, "net")
	if _, err := os.Lstat(netDir); err != nil {
		return nil, fmt.Errorf("GetNetName(): no net directory under pci device %s: %q", pciAddr, err)
	}

	fInfos, err := os.ReadDir(netDir)
	if err != nil {
		return nil, fmt.Errorf("GetNetName(): failed to read net directory %s: %q", netDir, err)
	}

	names := make([]string, 0)
	for _, f := range fInfos {
		names = append(names, f.Name())
	}

	return names, nil
}

// GetDriverName returns current driver attached to a pci device from its pci address
func GetDriverName(pciAddr string) (string, error) {
	driverLink := filepath.Join(sysBusPci, pciAddr, "driver")
	driverInfo, err := os.Readlink(driverLink)
	if err != nil {
		return "", fmt.Errorf("error getting driver info for device %s %v", pciAddr, err)
	}
	return filepath.Base(driverInfo), nil
}

// GetVFID returns VF ID index (within specific PF) based on PCI address
func GetVFID(pciAddr string) (vfID int, err error) {
	pfDir := filepath.Join(sysBusPci, pciAddr, "physfn")
	vfID = -1
	_, err = os.Lstat(pfDir)
	if os.IsNotExist(err) {
		return vfID, nil
	}
	if err != nil {
		err = fmt.Errorf("could not get PF directory information for VF device: %s, Err: %v", pciAddr, err)
		return vfID, err
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))
	if err != nil {
		err = fmt.Errorf("error reading VF directories %v", err)
		return vfID, err
	}

	// Read all VF directory and get VF ID
	for vfID := range vfDirs {
		dirN := fmt.Sprintf("%s/virtfn%d", pfDir, vfID)
		dirInfo, err := os.Lstat(dirN)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dirN)
			if err == nil && strings.Contains(linkName, pciAddr) {
				return vfID, err
			}
		}
	}
	// The requested VF not found
	vfID = -1
	return vfID, nil
}

// GetPfEswitchMode returns PF's eswitch mode for the given VF
// If device is not VF then it will return its own eswitch mode
func GetPfEswitchMode(pciAddr string) (string, error) {
	pfAddr, err := GetPfAddr(pciAddr)
	if err != nil {
		return "", fmt.Errorf("error getting PF PCI address for device %s %v", pciAddr, err)
	}
	devLinkDeviceAttrs, err := GetNetlinkProvider().GetDevLinkDeviceEswitchAttrs(pfAddr)
	if err != nil {
		return "", err
	}
	return devLinkDeviceAttrs.Mode, nil
}

// HasDefaultRoute returns true if PCI network device is default route interface
func HasDefaultRoute(pciAddr string) (bool, error) {
	// Get net interface name
	ifNames, err := GetNetNames(pciAddr)
	if err != nil {
		return false, fmt.Errorf("error trying get net device name for device %s", pciAddr)
	}

	if len(ifNames) > 0 { // there's at least one interface name found
		for _, ifName := range ifNames {
			routes, err := GetNetlinkProvider().GetIPv4RouteList(ifName) // IPv6 routes: all interface has at least one link local route entry
			if err != nil {
				glog.Errorf("failed to get routes for interface: %s, %q", ifName, err)
				continue
			}
			for _, r := range routes {
				if r.Dst == nil {
					glog.Infof("excluding interface %s:  default route found: %+v", ifName, r)
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// NormalizeVendorName returns vendor name cropped to fit into the length of maxVendorName if it is bigger
func NormalizeVendorName(vendor string) string {
	vendorName := vendor
	if len(vendor) > maxVendorName {
		vendorName = string([]byte(vendorName)[0:17]) + "..."
	}
	return vendorName
}

// NormalizeProductName returns product name cropped to fit into the length of maxProductName if it is bigger
func NormalizeProductName(product string) string {
	productName := product
	if len(product) > maxProductName {
		productName = string([]byte(productName)[0:37]) + "..."
	}
	return productName
}

// ParseDeviceID returns device ID parsed from the string as 64bit integer
func ParseDeviceID(deviceID string) (int64, error) {
	return strconv.ParseInt(deviceID, classIDBaseInt, classIDBitSize)
}

// ParseAuxDeviceType returns auxiliary device type parsed from device ID
func ParseAuxDeviceType(deviceID string) string {
	chunks := strings.Split(deviceID, ".")
	// auxiliary device name is of form <driver_name>.<kind_of_a_type>.<id> where id is an unsigned integer
	//nolint: gomnd
	if len(chunks) == 3 {
		if id, err := strconv.Atoi(chunks[2]); err == nil && id >= 0 {
			return chunks[1]
		}
	}
	// not an auxiliary device
	return ""
}
