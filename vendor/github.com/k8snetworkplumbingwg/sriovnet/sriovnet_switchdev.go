/*
Copyright 2023 NVIDIA CORPORATION & AFFILIATES

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sriovnet

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	utilfs "github.com/k8snetworkplumbingwg/sriovnet/pkg/utils/filesystem"
	"github.com/k8snetworkplumbingwg/sriovnet/pkg/utils/netlinkops"
)

const (
	netdevPhysSwitchID = "phys_switch_id"
	netdevPhysPortName = "phys_port_name"
)

type PortFlavour uint16

// Keep things consistent with netlink lib constants
// nolint:revive,stylecheck
const (
	PORT_FLAVOUR_PHYSICAL = iota
	PORT_FLAVOUR_CPU
	PORT_FLAVOUR_DSA
	PORT_FLAVOUR_PCI_PF
	PORT_FLAVOUR_PCI_VF
	PORT_FLAVOUR_VIRTUAL
	PORT_FLAVOUR_UNUSED
	PORT_FLAVOUR_PCI_SF
	PORT_FLAVOUR_UNKNOWN = 0xffff
)

// Regex that matches on the physical/upling port name
var physPortRepRegex = regexp.MustCompile(`^p(\d+)$`)

// Regex that matches on PF representor port name. These ports exists on DPUs.
var pfPortRepRegex = regexp.MustCompile(`^(?:c\d+)?pf(\d+)$`)

// Regex that matches on VF representor port name
var vfPortRepRegex = regexp.MustCompile(`^(?:c\d+)?pf(\d+)vf(\d+)$`)

// Regex that matches on SF representor port name
var sfPortRepRegex = regexp.MustCompile(`^(?:c\d+)?pf(\d+)sf(\d+)$`)

func parseIndexFromPhysPortName(portName string, regex *regexp.Regexp) (pfRepIndex, vfRepIndex int, err error) {
	pfRepIndex = -1
	vfRepIndex = -1

	matches := regex.FindStringSubmatch(portName)
	//nolint:gomnd
	if len(matches) != 3 {
		err = fmt.Errorf("failed to parse portName %s", portName)
	} else {
		pfRepIndex, err = strconv.Atoi(matches[1])
		if err == nil {
			vfRepIndex, err = strconv.Atoi(matches[2])
		}
	}
	return pfRepIndex, vfRepIndex, err
}

func parsePortName(physPortName string) (pfRepIndex, vfRepIndex int, err error) {
	// old kernel syntax of phys_port_name is vf index
	physPortName = strings.TrimSpace(physPortName)
	physPortNameInt, err := strconv.Atoi(physPortName)
	if err == nil {
		vfRepIndex = physPortNameInt
	} else {
		pfRepIndex, vfRepIndex, err = parseIndexFromPhysPortName(physPortName, vfPortRepRegex)
	}
	return pfRepIndex, vfRepIndex, err
}

func sfIndexFromPortName(physPortName string) (int, error) {
	//nolint:gomnd
	_, sfRepIndex, err := parseIndexFromPhysPortName(physPortName, sfPortRepRegex)
	return sfRepIndex, err
}

func isSwitchdev(netdevice string) bool {
	swIDFile := filepath.Join(NetSysDir, netdevice, netdevPhysSwitchID)
	physSwitchID, err := utilfs.Fs.ReadFile(swIDFile)
	if err != nil {
		return false
	}
	if len(physSwitchID) != 0 {
		return true
	}
	return false
}

// GetUplinkRepresentor gets a VF or PF PCI address (e.g '0000:03:00.4') and
// returns the uplink represntor netdev name for that VF or PF.
func GetUplinkRepresentor(pciAddress string) (string, error) {
	devicePath := filepath.Join(PciSysDir, pciAddress, "physfn", "net")
	if _, err := utilfs.Fs.Stat(devicePath); errors.Is(err, os.ErrNotExist) {
		// If physfn symlink to the parent PF doesn't exist, use the current device's dir
		devicePath = filepath.Join(PciSysDir, pciAddress, "net")
	}

	devices, err := utilfs.Fs.ReadDir(devicePath)
	if err != nil {
		return "", fmt.Errorf("failed to lookup %s: %v", pciAddress, err)
	}
	for _, device := range devices {
		if isSwitchdev(device.Name()) {
			// Try to get the phys port name, if not exists then fallback to check without it
			// phys_port_name should be in formant p<port-num> e.g p0,p1,p2 ...etc.
			if devicePhysPortName, err := getNetDevPhysPortName(device.Name()); err == nil {
				if !physPortRepRegex.MatchString(devicePhysPortName) {
					continue
				}
			}

			return device.Name(), nil
		}
	}
	return "", fmt.Errorf("uplink for %s not found", pciAddress)
}

func GetVfRepresentor(uplink string, vfIndex int) (string, error) {
	swIDFile := filepath.Join(NetSysDir, uplink, netdevPhysSwitchID)
	physSwitchID, err := utilfs.Fs.ReadFile(swIDFile)
	if err != nil || len(physSwitchID) == 0 {
		return "", fmt.Errorf("cant get uplink %s switch id", uplink)
	}

	pfSubsystemPath := filepath.Join(NetSysDir, uplink, "subsystem")
	devices, err := utilfs.Fs.ReadDir(pfSubsystemPath)
	if err != nil {
		return "", err
	}
	for _, device := range devices {
		devicePath := filepath.Join(NetSysDir, device.Name())
		deviceSwIDFile := filepath.Join(devicePath, netdevPhysSwitchID)
		deviceSwID, err := utilfs.Fs.ReadFile(deviceSwIDFile)
		if err != nil || !bytes.Equal(deviceSwID, physSwitchID) {
			continue
		}
		physPortNameStr, err := getNetDevPhysPortName(device.Name())
		if err != nil {
			continue
		}
		pfRepIndex, vfRepIndex, _ := parsePortName(physPortNameStr)
		if pfRepIndex != -1 {
			pfPCIAddress, err := getPCIFromDeviceName(uplink)
			if err != nil {
				continue
			}
			PCIFuncAddress, err := strconv.Atoi(string((pfPCIAddress[len(pfPCIAddress)-1])))
			if pfRepIndex != PCIFuncAddress || err != nil {
				continue
			}
		}
		// At this point we're confident we have a representor.
		if vfRepIndex == vfIndex {
			return device.Name(), nil
		}
	}
	return "", fmt.Errorf("failed to find VF representor for uplink %s", uplink)
}

func GetSfRepresentor(uplink string, sfNum int) (string, error) {
	pfNetPath := filepath.Join(NetSysDir, uplink, "device", "net")
	devices, err := utilfs.Fs.ReadDir(pfNetPath)
	if err != nil {
		return "", err
	}

	for _, device := range devices {
		physPortNameStr, err := getNetDevPhysPortName(device.Name())
		if err != nil {
			continue
		}
		sfRepIndex, err := sfIndexFromPortName(physPortNameStr)
		if err != nil {
			continue
		}
		if sfRepIndex == sfNum {
			return device.Name(), nil
		}
	}
	return "", fmt.Errorf("failed to find SF representor for uplink %s", uplink)
}

func getNetDevPhysPortName(netDev string) (string, error) {
	devicePortNameFile := filepath.Join(NetSysDir, netDev, netdevPhysPortName)
	physPortName, err := utilfs.Fs.ReadFile(devicePortNameFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(physPortName)), nil
}

// findNetdevWithPortNameCriteria returns representor netdev that matches a criteria function on the
// physical port name
func findNetdevWithPortNameCriteria(criteria func(string) bool) (string, error) {
	netdevs, err := utilfs.Fs.ReadDir(NetSysDir)
	if err != nil {
		return "", err
	}

	for _, netdev := range netdevs {
		// find matching VF representor
		netdevName := netdev.Name()

		// skip non switchdev netdevs
		if !isSwitchdev(netdevName) {
			continue
		}

		portName, err := getNetDevPhysPortName(netdevName)
		if err != nil {
			continue
		}

		if criteria(portName) {
			return netdevName, nil
		}
	}
	return "", fmt.Errorf("no representor matched criteria")
}

// GetPortIndexFromRepresentor finds the index of a representor from its network device name.
// Supports VF and SF. For multiple port flavors, the same ID could be returned, i.e.
//
//	pf0vf10 and pf0sf10
//
// will return the same port ID. To further differentiate the ports, use GetRepresentorPortFlavour
func GetPortIndexFromRepresentor(repNetDev string) (int, error) {
	flavor, err := GetRepresentorPortFlavour(repNetDev)
	if err != nil {
		return 0, err
	}

	if flavor != PORT_FLAVOUR_PCI_VF && flavor != PORT_FLAVOUR_PCI_SF {
		return 0, fmt.Errorf("unsupported port flavor for netdev %s", repNetDev)
	}

	physPortName, err := getNetDevPhysPortName(repNetDev)
	if err != nil {
		return 0, fmt.Errorf("failed to get device %s physical port name: %v", repNetDev, err)
	}

	typeToRegex := map[PortFlavour]*regexp.Regexp{
		PORT_FLAVOUR_PCI_VF: vfPortRepRegex,
		PORT_FLAVOUR_PCI_SF: sfPortRepRegex,
	}

	_, repIndex, err := parseIndexFromPhysPortName(physPortName, typeToRegex[flavor])
	if err != nil {
		return 0, fmt.Errorf("failed to parse the physical port name of device %s: %v", repNetDev, err)
	}

	return repIndex, nil
}

// GetVfRepresentorDPU returns VF representor on DPU for a host VF identified by pfID and vfIndex
func GetVfRepresentorDPU(pfID, vfIndex string) (string, error) {
	// TODO(Adrianc): This method should change to get switchID and vfIndex as input, then common logic can
	// be shared with GetVfRepresentor, backward compatibility should be preserved when this happens.

	// pfID should be 0 or 1
	if pfID != "0" && pfID != "1" {
		return "", fmt.Errorf("unexpected pfID(%s). It should be 0 or 1", pfID)
	}

	// vfIndex should be an unsinged integer provided as a decimal number
	if _, err := strconv.ParseUint(vfIndex, 10, 32); err != nil {
		return "", fmt.Errorf("unexpected vfIndex(%s). It should be an unsigned decimal number", vfIndex)
	}

	// map for easy search of expected VF rep port name.
	// Note: no support for Multi-Chassis DPUs
	expectedPhysPortNames := map[string]interface{}{
		fmt.Sprintf("pf%svf%s", pfID, vfIndex):   nil,
		fmt.Sprintf("c1pf%svf%s", pfID, vfIndex): nil,
	}

	netdev, err := findNetdevWithPortNameCriteria(func(portName string) bool {
		// if phys port name == pf<pfIndex>vf<vfIndex> or c1pf<pfIndex>vf<vfIndex> we have a match
		if _, ok := expectedPhysPortNames[portName]; ok {
			return true
		}
		return false
	})

	if err != nil {
		return "", fmt.Errorf("vf representor for pfID:%s, vfIndex:%s not found", pfID, vfIndex)
	}
	return netdev, nil
}

// GetSfRepresentorDPU returns SF representor on DPU for a host SF identified by pfID and sfIndex
func GetSfRepresentorDPU(pfID, sfIndex string) (string, error) {
	// pfID should be 0 or 1
	if pfID != "0" && pfID != "1" {
		return "", fmt.Errorf("unexpected pfID(%s). It should be 0 or 1", pfID)
	}

	// sfIndex should be an unsinged integer provided as a decimal number
	if _, err := strconv.ParseUint(sfIndex, 10, 32); err != nil {
		return "", fmt.Errorf("unexpected sfIndex(%s). It should be an unsigned decimal number", sfIndex)
	}

	// map for easy search of expected VF rep port name.
	// Note: no support for Multi-Chassis DPUs
	expectedPhysPortNames := map[string]interface{}{
		fmt.Sprintf("pf%ssf%s", pfID, sfIndex):   nil,
		fmt.Sprintf("c1pf%ssf%s", pfID, sfIndex): nil,
	}

	netdev, err := findNetdevWithPortNameCriteria(func(portName string) bool {
		// if phys port name == pf<pfIndex>sf<sfIndex> or c1pf<pfIndex>sf<sfIndex> we have a match
		if _, ok := expectedPhysPortNames[portName]; ok {
			return true
		}
		return false
	})

	if err != nil {
		return "", fmt.Errorf("sf representor for pfID:%s, sfIndex:%s not found", pfID, sfIndex)
	}
	return netdev, nil
}

// GetRepresentorPortFlavour returns the representor port flavour
// Note: this method does not support old representor names used by old kernels
// e.g <vf_num> and will return PORT_FLAVOUR_UNKNOWN for such cases.
func GetRepresentorPortFlavour(netdev string) (PortFlavour, error) {
	if !isSwitchdev(netdev) {
		return PORT_FLAVOUR_UNKNOWN, fmt.Errorf("net device %s is does not represent an eswitch port", netdev)
	}

	// Attempt to get information via devlink (Kernel >= 5.9.0)
	port, err := netlinkops.GetNetlinkOps().DevLinkGetPortByNetdevName(netdev)
	if err == nil {
		return PortFlavour(port.PortFlavour), nil
	}

	// Fallback to Get PortFlavour by phys_port_name
	// read phy_port_name
	portName, err := getNetDevPhysPortName(netdev)
	if err != nil {
		return PORT_FLAVOUR_UNKNOWN, err
	}

	typeToRegex := map[PortFlavour]*regexp.Regexp{
		PORT_FLAVOUR_PHYSICAL: physPortRepRegex,
		PORT_FLAVOUR_PCI_PF:   pfPortRepRegex,
		PORT_FLAVOUR_PCI_VF:   vfPortRepRegex,
		PORT_FLAVOUR_PCI_SF:   sfPortRepRegex,
	}
	for flavour, regex := range typeToRegex {
		if regex.MatchString(portName) {
			return flavour, nil
		}
	}
	return PORT_FLAVOUR_UNKNOWN, nil
}

// parseDPUConfigFileOutput parses the config file content of a DPU
// representor port. The format of the file is a set of <key>:<value> pairs as follows:
//
// ```
//
//	MAC        : 0c:42:a1:c6:cf:7c
//	MaxTxRate  : 0
//	State      : Follow
//
// ```
func parseDPUConfigFileOutput(out string) map[string]string {
	configMap := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		entry := strings.SplitN(line, ":", 2)
		if len(entry) != 2 {
			// unexpected line format
			continue
		}
		configMap[strings.Trim(entry[0], " \t\n")] = strings.Trim(entry[1], " \t\n")
	}
	return configMap
}

// GetRepresentorPeerMacAddress returns the MAC address of the peer netdev associated with the given
// representor netdev
// Note:
//
//	This method functionality is currently supported only on DPUs.
//	Currently only netdev representors with PORT_FLAVOUR_PCI_PF are supported
func GetRepresentorPeerMacAddress(netdev string) (net.HardwareAddr, error) {
	flavor, err := GetRepresentorPortFlavour(netdev)
	if err != nil {
		return nil, fmt.Errorf("unknown port flavour for netdev %s. %v", netdev, err)
	}
	if flavor == PORT_FLAVOUR_UNKNOWN {
		return nil, fmt.Errorf("unknown port flavour for netdev %s", netdev)
	}
	if flavor != PORT_FLAVOUR_PCI_PF {
		return nil, fmt.Errorf("unsupported port flavour for netdev %s", netdev)
	}

	// Attempt to get information via devlink (Kernel >= 5.9.0)
	port, err := netlinkops.GetNetlinkOps().DevLinkGetPortByNetdevName(netdev)
	if err == nil {
		if port.Fn != nil {
			return port.Fn.HwAddr, nil
		}
	}

	// Get information via sysfs
	// read phy_port_name
	portName, err := getNetDevPhysPortName(netdev)
	if err != nil {
		return nil, err
	}
	// Extract port num
	portNum := pfPortRepRegex.FindStringSubmatch(portName)
	if len(portNum) < 2 {
		return nil, fmt.Errorf("failed to extract physical port number from port name %s of netdev %s",
			portName, netdev)
	}
	uplinkPhysPortName := "p" + portNum[1]
	// Find uplink netdev for that port
	// Note(adrianc): As we support only DPUs ATM we do not need to deal with netdevs from different
	// eswitch (i.e different switch IDs).
	uplinkNetdev, err := findNetdevWithPortNameCriteria(func(pname string) bool { return pname == uplinkPhysPortName })
	if err != nil {
		return nil, fmt.Errorf("failed to find uplink port for netdev %s. %v", netdev, err)
	}
	// get MAC address for netdev
	configPath := filepath.Join(NetSysDir, uplinkNetdev, "smart_nic", "pf", "config")
	out, err := utilfs.Fs.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DPU config via uplink %s for %s. %v",
			uplinkNetdev, netdev, err)
	}
	config := parseDPUConfigFileOutput(string(out))
	macStr, ok := config["MAC"]
	if !ok {
		return nil, fmt.Errorf("MAC address not found for %s", netdev)
	}
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse MAC address \"%s\" for %s. %v", macStr, netdev, err)
	}
	return mac, nil
}

// SetRepresentorPeerMacAddress sets the given MAC addresss of the peer netdev associated with the given
// representor netdev.
// Note: This method functionality is currently supported only for DPUs.
// Currently only netdev representors with PORT_FLAVOUR_PCI_VF are supported
func SetRepresentorPeerMacAddress(netdev string, mac net.HardwareAddr) error {
	flavor, err := GetRepresentorPortFlavour(netdev)
	if err != nil {
		return fmt.Errorf("unknown port flavour for netdev %s. %v", netdev, err)
	}
	if flavor == PORT_FLAVOUR_UNKNOWN {
		return fmt.Errorf("unknown port flavour for netdev %s", netdev)
	}
	if flavor != PORT_FLAVOUR_PCI_VF {
		return fmt.Errorf("unsupported port flavour for netdev %s", netdev)
	}

	physPortNameStr, err := getNetDevPhysPortName(netdev)
	if err != nil {
		return fmt.Errorf("failed to get phys_port_name for netdev %s: %v", netdev, err)
	}
	pfID, vfIndex, err := parsePortName(physPortNameStr)
	if err != nil {
		return fmt.Errorf("failed to get the pf and vf index for netdev %s "+
			"with phys_port_name %s: %v", netdev, physPortNameStr, err)
	}

	uplinkPhysPortName := fmt.Sprintf("p%d", pfID)
	uplinkNetdev, err := findNetdevWithPortNameCriteria(func(pname string) bool { return pname == uplinkPhysPortName })
	if err != nil {
		return fmt.Errorf("failed to find netdev for physical port name %s. %v", uplinkPhysPortName, err)
	}
	vfRepName := fmt.Sprintf("vf%d", vfIndex)
	sysfsVfRepMacFile := filepath.Join(NetSysDir, uplinkNetdev, "smart_nic", vfRepName, "mac")
	_, err = utilfs.Fs.Stat(sysfsVfRepMacFile)
	if err != nil {
		return fmt.Errorf("couldn't stat VF representor's sysfs file %s: %v", sysfsVfRepMacFile, err)
	}
	err = utilfs.Fs.WriteFile(sysfsVfRepMacFile, []byte(mac.String()), 0)
	if err != nil {
		return fmt.Errorf("failed to write the MAC address %s to VF reprentor %s",
			mac.String(), sysfsVfRepMacFile)
	}
	return nil
}
