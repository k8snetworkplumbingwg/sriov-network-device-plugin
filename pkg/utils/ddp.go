package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// /*
// Example output of DDP tool
// $ ./ddptool -l -a -j -s 0000:02:00.0
// {
//         "DDPInventory": {
//                 "device": "1572",
//                 "address": "0000:02:00.0",
//                 "name": "enp2s0f0",
//                 "display": "Intel(R) Ethernet Converged Network Adapter X710-4",
//                 "DDPpackage": {
//                         "track_id": "80000008",
//                         "version": "1.0.3.0",
//                         "name": "GTPv1-C/U IPv4/IPv6 payload"
//                 }
//         }
// }

// */

// DDPInfo is the toplevel container of DDPInventory
type DDPInfo struct {
	DDPInventory DDPInventory `json:"DDPInventory"`
}

// DDPInventory holds a device's DDP information
type DDPInventory struct {
	Device     string     `json:"device"`
	Address    string     `json:"address"`
	Name       string     `json:"name"`
	Display    string     `json:"display"`
	DDPpackage DDPpackage `json:"DDPpackage"`
}

// DDPpackage holds information about DDP profile itself
type DDPpackage struct {
	TrackID string `json:"track_id"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

// 8 is an exit code of ddptool when profile was not found
const ddpNoDDPProfile = 8

var ddpExecCommand = exec.Command

// IsDDPToolSupportedByDevice checks if DDPTool can be used with device
func IsDDPToolSupportedByDevice(dev string) bool {
	if _, err := GetDDPProfiles(dev); err != nil && !errors.Is(err, ErrProfileNameNotFound) {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == ddpNoDDPProfile {
				return true
			}
		}

		return false
	}

	return true
}

// GetDDPProfiles returns running DDP profile name if available
func GetDDPProfiles(dev string) (string, error) {
	var stdout bytes.Buffer
	cmd := ddpExecCommand("ddptool", "-l", "-a", "-j", "-s", dev)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if strings.Contains(err.Error(), ErrProfileNameNotFound.Error()) {
			return "", fmt.Errorf("error while getting DDP profiles: %w", ErrProfileNameNotFound)
		}
		return "", err
	}

	return getDDPNameFromStdout(stdout.Bytes())
}

// ErrProfileNameNotFound error when DDPTool is supported, but package name is empty
var ErrProfileNameNotFound = errors.New("DDP profile name not found")

func getDDPNameFromStdout(in []byte) (string, error) {
	ddpInfo := &DDPInfo{}
	if err := json.Unmarshal(in, ddpInfo); err != nil {
		return "", err
	}

	if ddpInfo.DDPInventory.DDPpackage.Name == "" {
		return "", fmt.Errorf("DDP profile name not found")
	}

	return ddpInfo.DDPInventory.DDPpackage.Name, nil
}
