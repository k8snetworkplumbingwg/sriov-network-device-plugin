package resources

import (
	"fmt"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"strconv"
	"strings"
)

// newVendorSelector returns a DeviceSelector interface for vendor list
func newVendorSelector(vendors []string) types.DeviceSelector {
	return &vendorSelector{vendors: vendors}
}

type vendorSelector struct {
	vendors []string
}

func (s *vendorSelector) Filter(inDevices []types.PciNetDevice) []types.PciNetDevice {
	filteredList := make([]types.PciNetDevice, 0)
	for _, dev := range inDevices {
		devVendor := dev.GetVendor()
		if contains(s.vendors, devVendor) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// newDeviceSelector returns a DeviceSelector interface for device list
func newDeviceSelector(devices []string) types.DeviceSelector {
	return &deviceSelector{devices: devices}
}

type deviceSelector struct {
	devices []string
}

func (s *deviceSelector) Filter(inDevices []types.PciNetDevice) []types.PciNetDevice {
	filteredList := make([]types.PciNetDevice, 0)
	for _, dev := range inDevices {
		devCode := dev.GetDeviceCode()
		if contains(s.devices, devCode) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// newDriverSelector returns a DeviceSelector interface for driver list
func newDriverSelector(drivers []string) types.DeviceSelector {
	return &driverSelector{drivers: drivers}
}

type driverSelector struct {
	drivers []string
}

func (s *driverSelector) Filter(inDevices []types.PciNetDevice) []types.PciNetDevice {
	filteredList := make([]types.PciNetDevice, 0)
	for _, dev := range inDevices {
		devDriver := dev.GetDriver()
		if contains(s.drivers, devDriver) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// newPfNameSelector returns a NetDevSelector interface for netDev list
func newPfNameSelector(pfNames []string) types.DeviceSelector {
	return &pfNameSelector{pfNames: pfNames}
}

type pfNameSelector struct {
	pfNames []string
}

func (s *pfNameSelector) Filter(inDevices []types.PciNetDevice) []types.PciNetDevice {
	filteredList := make([]types.PciNetDevice, 0)
	for _, dev := range inDevices {
		selector := getItem(s.pfNames, dev.GetPFName())
		if selector != "" {
			if strings.Contains(selector, "#") {
				// Selector does contain VF index in next format:
				// <PFName>#<VFIndexStart>-<VFIndexEnd>
				// In this case both <VFIndexStart> and <VFIndexEnd>
				// are included in range, for example: "netpf0#3-5"
				// The VFs 3,4 and 5 of the PF 'netpf0' will be included
				// in selector pool
				fields := strings.Split(selector, "#")
				if len(fields) != 2 {
					fmt.Printf("Failed to parse %s PF name selector, probably incorrect separator character usage\n", dev.GetPFName())
					continue
				}
				rng := strings.Split(fields[1], "-")
				if len(rng) != 2 {
					fmt.Printf("Failed to parse %s PF name selector, probably incorrect range character usage\n", dev.GetPFName())
					continue
				}
				rngSt, err := strconv.Atoi(rng[0])
				if err != nil {
					fmt.Printf("Failed to parse %s PF name selector, start range is incorrect\n", dev.GetPFName())
					continue
				}
				rngEnd, err := strconv.Atoi(rng[1])
				if err != nil {
					fmt.Printf("Failed to parse %s PF name selector, end range is incorrect\n", dev.GetPFName())
					continue
				}
				vfID := dev.GetVFID()
				if vfID >= rngSt && vfID <= rngEnd {
					filteredList = append(filteredList, dev)
				}
			} else {
				filteredList = append(filteredList, dev)
			}
		}
	}

	return filteredList
}

// newLinkTypeSelector returns a interface for netDev list
func newLinkTypeSelector(linkTypes []string) types.DeviceSelector {
	return &linkTypeSelector{linkTypes: linkTypes}
}

type linkTypeSelector struct {
	linkTypes []string
}

func (s *linkTypeSelector) Filter(inDevices []types.PciNetDevice) []types.PciNetDevice {
	filteredList := make([]types.PciNetDevice, 0)
	for _, dev := range inDevices {
		if contains(s.linkTypes, dev.GetLinkType()) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

func getItem(hay []string, needle string) string {
	for _, item := range hay {
		if strings.HasPrefix(item, needle) {
			return item
		}
	}
	return ""
}
