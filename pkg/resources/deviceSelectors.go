package resources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

const (
	rngSplitTotal   = 2
	fieldSplitTotal = 2
)

// NewVendorSelector returns a DeviceSelector interface for vendor list
func NewVendorSelector(vendors []string) types.DeviceSelector {
	return &vendorSelector{vendors: vendors}
}

type vendorSelector struct {
	vendors []string
}

func (s *vendorSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		devVendor := dev.GetVendor()
		if contains(s.vendors, devVendor) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewDeviceSelector returns a DeviceSelector interface for device list
func NewDeviceSelector(devices []string) types.DeviceSelector {
	return &deviceSelector{devices: devices}
}

type deviceSelector struct {
	devices []string
}

func (s *deviceSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		devCode := dev.GetDeviceCode()
		if contains(s.devices, devCode) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewDriverSelector returns a DeviceSelector interface for driver list
func NewDriverSelector(drivers []string) types.DeviceSelector {
	return &driverSelector{drivers: drivers}
}

type driverSelector struct {
	drivers []string
}

func (s *driverSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		devDriver := dev.GetDriver()
		if contains(s.drivers, devDriver) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewPciAddressSelector returns a NetDevSelector interface for netDev list
func NewPciAddressSelector(pciAddresses []string) types.DeviceSelector {
	return &pciAddressSelector{pciAddresses: pciAddresses}
}

type pciAddressSelector struct {
	pciAddresses []string
}

func (s *pciAddressSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		pciAddr := dev.(types.PciDevice).GetPciAddr()
		if contains(s.pciAddresses, pciAddr) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewPfNameSelector returns a NetDevSelector interface for netDev list
func NewPfNameSelector(pfNames []string) types.DeviceSelector {
	return &pfNameSelector{pfNames: pfNames}
}

type pfNameSelector struct {
	pfNames []string
}

func (s *pfNameSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		pfName := dev.(types.NetDevice).GetPfNetName()
		if pfName == "" {
			// Exclude devices that doesn't have a PF name
			continue
		}
		devIdx := dev.(types.NetDevice).GetFuncID()
		selector := getItem(s.pfNames, pfName)
		if selector != "" {
			if isSelected(devIdx, selector) {
				filteredList = append(filteredList, dev)
			}
		}
	}

	return filteredList
}

// NewNicNameSelector returns a DeviceSelector interface for nicName list
func NewNicNameSelector(nicNames []string) types.DeviceSelector {
	return &nicNameSelector{nicNames: nicNames}
}

type nicNameSelector struct {
	nicNames []string
}

func (s *nicNameSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		nicName := dev.(types.NetDevice).GetNetName()
		if contains(s.nicNames, nicName) {
			filteredList = append(filteredList, dev)
		}
	}

	return filteredList
}

// NewRootDeviceSelector returns a NetDevSelector interface for netDev list
func NewRootDeviceSelector(rootDevices []string) types.DeviceSelector {
	return &rootDeviceSelector{rootDevices: rootDevices}
}

type rootDeviceSelector struct {
	rootDevices []string
}

func (s *rootDeviceSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		rootDevice := dev.(types.NetDevice).GetPfPciAddr()
		if rootDevice == "" {
			// Exclude devices that doesn't have a root PCI device
			continue
		}
		devIdx := dev.(types.NetDevice).GetFuncID()
		selector := getItem(s.rootDevices, rootDevice)
		if selector != "" {
			if isSelected(devIdx, selector) {
				filteredList = append(filteredList, dev)
			}
		}
	}
	return filteredList
}

// NewLinkTypeSelector returns a interface for netDev list
func NewLinkTypeSelector(linkTypes []string) types.DeviceSelector {
	return &linkTypeSelector{linkTypes: linkTypes}
}

type linkTypeSelector struct {
	linkTypes []string
}

func (s *linkTypeSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		linkType := dev.(types.NetDevice).GetLinkType()
		if contains(s.linkTypes, linkType) {
			filteredList = append(filteredList, dev)
		}
	}
	return filteredList
}

// NewAuxTypeSelector returns an interface for auxTypes list
func NewAuxTypeSelector(auxTypes []string) types.DeviceSelector {
	return &auxTypeSelector{auxTypes: auxTypes}
}

type auxTypeSelector struct {
	auxTypes []string
}

func (s *auxTypeSelector) Filter(inDevices []types.HostDevice) []types.HostDevice {
	filteredList := make([]types.HostDevice, 0)
	for _, dev := range inDevices {
		auxType := dev.(types.AuxNetDevice).GetAuxType()
		if contains(s.auxTypes, auxType) {
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
		if strings.EqualFold(strings.Split(item, "#")[0], needle) {
			return item
		}
	}
	return ""
}

func isSelected(devIdx int, selector string) bool {
	if strings.Contains(selector, "#") {
		// Selector does contain index in next format:
		// <PFName>#<IndexStart>-<IndexEnd> or
		// <PFAddr>#<IndexStart>-<IndexEnd>
		// In this case both <IndexStart> and <IndexEnd>
		// are included in range, for example: "netpf0#3-5"
		// Functions 3,4 and 5 of the PF 'netpf0' will be included
		// in selector pool
		fields := strings.Split(selector, "#")
		if len(fields) != fieldSplitTotal {
			fmt.Printf("Failed to parse %s PF (name|address) selector, probably incorrect separator character usage\n", selector)
			return false
		}
		entries := strings.Split(fields[1], ",")
		for i := 0; i < len(entries); i++ {
			if strings.Contains(entries[i], "-") {
				rng := strings.Split(entries[i], "-")
				if len(rng) != rngSplitTotal {
					fmt.Printf("Failed to parse %s PF (name|address) selector, probably incorrect range character usage\n", selector)
					return false
				}
				rngSt, err := strconv.Atoi(rng[0])
				if err != nil {
					fmt.Printf("Failed to parse %s PF (name|address) selector, start range is incorrect\n", selector)
					return false
				}
				rngEnd, err := strconv.Atoi(rng[1])
				if err != nil {
					fmt.Printf("Failed to parse %s PF (name|address) selector, end range is incorrect\n", selector)
					return false
				}
				if devIdx >= rngSt && devIdx <= rngEnd {
					return true
				}
			} else {
				funcid, err := strconv.Atoi(entries[i])
				if err != nil {
					fmt.Printf("Failed to parse %s PF (name|address) selector, index is incorrect\n", selector)
					return false
				}
				if devIdx == funcid {
					return true
				}
			}
		}
	} else {
		return true
	}
	return false
}
