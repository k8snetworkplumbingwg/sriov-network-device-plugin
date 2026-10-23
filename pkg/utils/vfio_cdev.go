// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const vfioNoIOMMUGroup = "vfio-noiommu"

// ResolveVFIOCdev resolves and validates the native VFIO node for a PCI function.
// hostDevDir is an inspection mount. The returned path is always in host /dev.
func ResolveVFIOCdev(pciPath, hostDevDir string) (string, error) {
	driver, err := os.Readlink(filepath.Join(pciPath, "driver"))
	if err != nil {
		return "", fmt.Errorf("read VFIO driver: %w", err)
	}
	if filepath.Base(driver) != "vfio-pci" {
		return "", fmt.Errorf("device is not bound to vfio-pci")
	}
	nameBytes, nameErr := os.ReadFile(filepath.Join(pciPath, "iommu_group", "name"))
	if nameErr == nil && strings.TrimSpace(string(nameBytes)) == vfioNoIOMMUGroup {
		return "", fmt.Errorf("native VFIO does not support no-IOMMU devices")
	}
	entries, err := os.ReadDir(filepath.Join(pciPath, "vfio-dev"))
	if err != nil {
		return "", fmt.Errorf("resolve VFIO cdev: %w", err)
	}
	if len(entries) != 1 {
		return "", fmt.Errorf("expected one VFIO cdev, found %d", len(entries))
	}
	name := entries[0].Name()
	if !strings.HasPrefix(name, "vfio") {
		return "", fmt.Errorf("invalid VFIO cdev name %q", name)
	}
	if _, err := strconv.ParseUint(strings.TrimPrefix(name, "vfio"), 10, 32); err != nil {
		return "", fmt.Errorf("invalid VFIO cdev name %q", name)
	}
	expected, err := os.ReadFile(filepath.Join(pciPath, "vfio-dev", name, "dev"))
	if err != nil {
		return "", fmt.Errorf("read VFIO device number: %w", err)
	}
	if err := validateDeviceNumber(filepath.Join(hostDevDir, "vfio", "devices", name), string(expected)); err != nil {
		return "", err
	}
	return filepath.Join("/dev/vfio/devices", name), nil
}

func validateDeviceNumber(node, expected string) error {
	majorText, minorText, ok := strings.Cut(strings.TrimSpace(expected), ":")
	if !ok {
		return fmt.Errorf("invalid device number %q", expected)
	}
	major, err := strconv.ParseUint(majorText, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid major number: %w", err)
	}
	minor, err := strconv.ParseUint(minorText, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid minor number: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Stat(node, &st); err != nil {
		return fmt.Errorf("inspect %s: %w", node, err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFCHR {
		return fmt.Errorf("%s is not a character device", node)
	}
	if uint64(unix.Major(st.Rdev)) != major || uint64(unix.Minor(st.Rdev)) != minor {
		return fmt.Errorf("%s does not match sysfs device number %s", node, strings.TrimSpace(expected))
	}
	return nil
}

// ValidateIOMMUDevice validates the control node against the host sysfs identity.
func ValidateIOMMUDevice(hostDevDir string) error {
	expected, err := os.ReadFile(filepath.Join(sysClassIommu, "dev"))
	if err != nil {
		return fmt.Errorf("read IOMMU device number: %w", err)
	}
	return validateDeviceNumber(filepath.Join(hostDevDir, "iommu"), string(expected))
}

// GetVFIOCdevFile returns the validated native VFIO node for a PCI address.
func GetVFIOCdevFile(pciAddr, hostDevDir string) (string, error) {
	return ResolveVFIOCdev(filepath.Join(sysBusPci, pciAddr), hostDevDir)
}
