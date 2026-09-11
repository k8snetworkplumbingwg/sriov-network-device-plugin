/*
 * SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package resources

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const vfioPCIDriver = "vfio-pci"

type pciDriverManager interface {
	EnsureDrivers(pciAddrs []string, desiredDriver string) error
}

type pciDriverOps interface {
	CurrentDriver(pciAddr string) (string, error)
	DriverExists(driver string) bool
	DeviceInUse(pciAddr string) (bool, string, error)
	SetDriverOverride(pciAddr, driver string) error
	Unbind(pciAddr, driver string) error
	Bind(pciAddr, driver string) error
}

type hostPCIDriverManager struct {
	ops   pciDriverOps
	locks sync.Map
}

func newHostPCIDriverManager() pciDriverManager {
	return &hostPCIDriverManager{
		ops: &hostPCIDriverOps{
			pciBusPath:  "/host/sys/bus/pci",
			procPath:    "/host/proc",
			vfioDevPath: "/host/dev/vfio",
		},
	}
}

func (m *hostPCIDriverManager) EnsureDrivers(pciAddrs []string, desiredDriver string) error {
	addresses := uniqueSorted(pciAddrs)
	locks := make([]*sync.Mutex, 0, len(addresses))
	for _, pciAddr := range addresses {
		deviceLock, _ := m.locks.LoadOrStore(pciAddr, &sync.Mutex{})
		lock := deviceLock.(*sync.Mutex)
		lock.Lock()
		locks = append(locks, lock)
	}
	defer func() {
		for index := len(locks) - 1; index >= 0; index-- {
			locks[index].Unlock()
		}
	}()

	for _, pciAddr := range addresses {
		if _, err := m.preflight(pciAddr, desiredDriver); err != nil {
			return err
		}
	}
	for _, pciAddr := range addresses {
		// Kata or another host process does not participate in the locks above.
		// Revalidate immediately before mutation instead of acting on the batch
		// preflight snapshot.
		currentDriver, err := m.preflight(pciAddr, desiredDriver)
		if err != nil {
			return err
		}
		if err := m.ensureDriver(pciAddr, currentDriver, desiredDriver); err != nil {
			return err
		}
	}
	return nil
}

func (m *hostPCIDriverManager) preflight(pciAddr, desiredDriver string) (string, error) {
	currentDriver, err := m.ops.CurrentDriver(pciAddr)
	if err != nil {
		return "", fmt.Errorf("get current driver for PCI device %s: %w", pciAddr, err)
	}

	if currentDriver == desiredDriver {
		return currentDriver, nil
	}
	if currentDriver != vfioPCIDriver {
		return "", fmt.Errorf("PCI device %s uses driver %q; expected %q or %q",
			pciAddr, currentDriver, desiredDriver, vfioPCIDriver)
	}
	if !m.ops.DriverExists(desiredDriver) {
		return "", fmt.Errorf("desired driver %q is not loaded for PCI device %s", desiredDriver, pciAddr)
	}

	inUse, owner, err := m.ops.DeviceInUse(pciAddr)
	if err != nil {
		return "", fmt.Errorf("determine VFIO ownership for PCI device %s: %w", pciAddr, err)
	}
	if inUse {
		return "", fmt.Errorf("PCI device %s is still in use by %s", pciAddr, owner)
	}
	return currentDriver, nil
}

func (m *hostPCIDriverManager) ensureDriver(pciAddr, currentDriver, desiredDriver string) error {
	if currentDriver == desiredDriver {
		if err := m.ops.SetDriverOverride(pciAddr, ""); err != nil {
			return fmt.Errorf("clear driver override for PCI device %s: %w", pciAddr, err)
		}
		return m.verifyDriver(pciAddr, desiredDriver, "desired")
	}
	if err := m.ops.SetDriverOverride(pciAddr, desiredDriver); err != nil {
		return fmt.Errorf("set driver override for PCI device %s to %q: %w", pciAddr, desiredDriver, err)
	}
	// Narrow the external race between preflight and unbind. In particular,
	// do not enter the VFIO unregister wait if a process acquired the device
	// after the batch preflight.
	if inUse, owner, err := m.ops.DeviceInUse(pciAddr); err != nil {
		clearErr := m.ops.SetDriverOverride(pciAddr, "")
		return errors.Join(
			fmt.Errorf("recheck VFIO ownership for PCI device %s: %w", pciAddr, err),
			wrapCleanupError("clear driver override", clearErr),
		)
	} else if inUse {
		clearErr := m.ops.SetDriverOverride(pciAddr, "")
		return errors.Join(
			fmt.Errorf("PCI device %s is still in use by %s", pciAddr, owner),
			wrapCleanupError("clear driver override", clearErr),
		)
	}
	if err := m.ops.Unbind(pciAddr, currentDriver); err != nil {
		clearErr := m.ops.SetDriverOverride(pciAddr, "")
		return errors.Join(
			fmt.Errorf("unbind PCI device %s from %q: %w", pciAddr, currentDriver, err),
			wrapCleanupError("clear driver override", clearErr),
		)
	}

	if err := m.ops.Bind(pciAddr, desiredDriver); err != nil {
		rollbackErr := m.rollback(pciAddr, currentDriver)
		return errors.Join(
			fmt.Errorf("bind PCI device %s to desired driver %q: %w", pciAddr, desiredDriver, err),
			wrapCleanupError("rollback to vfio-pci", rollbackErr),
		)
	}

	if err := m.ops.SetDriverOverride(pciAddr, ""); err != nil {
		return fmt.Errorf("clear driver override for PCI device %s: %w", pciAddr, err)
	}
	return m.verifyDriver(pciAddr, desiredDriver, "desired")
}

func (m *hostPCIDriverManager) verifyDriver(pciAddr, expectedDriver, description string) error {
	boundDriver, err := m.ops.CurrentDriver(pciAddr)
	if err != nil {
		return fmt.Errorf("verify %s driver for PCI device %s: %w", description, pciAddr, err)
	}
	if boundDriver != expectedDriver {
		return fmt.Errorf("PCI device %s was bound to %q instead of %s driver %q",
			pciAddr, boundDriver, description, expectedDriver)
	}
	return nil
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (m *hostPCIDriverManager) rollback(pciAddr, driver string) error {
	setErr := m.ops.SetDriverOverride(pciAddr, driver)
	if setErr != nil {
		return setErr
	}
	bindErr := m.ops.Bind(pciAddr, driver)
	clearErr := m.ops.SetDriverOverride(pciAddr, "")
	boundDriver, currentErr := m.ops.CurrentDriver(pciAddr)
	if currentErr == nil && boundDriver != driver {
		currentErr = fmt.Errorf("PCI device %s was bound to %q instead of rollback driver %q",
			pciAddr, boundDriver, driver)
	}
	return errors.Join(bindErr, clearErr, currentErr)
}

func wrapCleanupError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}

type hostPCIDriverOps struct {
	pciBusPath  string
	procPath    string
	vfioDevPath string
}

type vfioFileIdentity struct {
	info os.FileInfo
}

func (o *hostPCIDriverOps) CurrentDriver(pciAddr string) (string, error) {
	driverLink, err := os.Readlink(filepath.Join(o.pciBusPath, "devices", pciAddr, "driver"))
	if err != nil {
		return "", err
	}
	return filepath.Base(driverLink), nil
}

func (o *hostPCIDriverOps) DriverExists(driver string) bool {
	info, err := os.Stat(filepath.Join(o.pciBusPath, "drivers", driver))
	return err == nil && info.IsDir()
}

func (o *hostPCIDriverOps) DeviceInUse(pciAddr string) (bool, string, error) {
	deviceNodes, err := o.vfioDeviceNodes(pciAddr)
	if err != nil {
		return false, "", err
	}
	identities := make([]vfioFileIdentity, 0, len(deviceNodes))
	for _, path := range deviceNodes {
		info, err := os.Stat(path)
		if err != nil {
			return false, "", fmt.Errorf("stat VFIO device node %s: %w", path, err)
		}
		identities = append(identities, vfioFileIdentity{info: info})
	}

	processes, err := os.ReadDir(o.procPath)
	if err != nil {
		return false, "", err
	}
	for _, process := range processes {
		if !process.IsDir() || !isDecimal(process.Name()) {
			continue
		}
		inUse, err := o.processUsesVFIO(process.Name(), pciAddr, identities)
		if err != nil {
			return false, "", err
		}
		if inUse {
			return true, o.processDescription(process.Name()), nil
		}
	}
	return false, "", nil
}

func (o *hostPCIDriverOps) processUsesVFIO(pid, pciAddr string, identities []vfioFileIdentity) (bool, error) {
	fdDir := filepath.Join(o.procPath, pid, "fd")
	fds, err := os.ReadDir(fdDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("inspect process %s file descriptors: %w", pid, err)
	}
	for _, fd := range fds {
		matches, err := o.fdUsesVFIO(pid, fd.Name(), pciAddr, identities)
		if err != nil || matches {
			return matches, err
		}
	}
	return false, nil
}

func (o *hostPCIDriverOps) fdUsesVFIO(pid, fd, pciAddr string, identities []vfioFileIdentity) (bool, error) {
	fdPath := filepath.Join(o.procPath, pid, "fd", fd)
	linkTarget, err := os.Readlink(fdPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect process %s file descriptor %s link: %w", pid, fd, err)
	}
	if linkTarget == "anon_inode:[vfio-device]" {
		matches, identified, err := o.anonymousVFIODeviceMatches(pid, fd, pciAddr)
		if err != nil {
			return false, err
		}
		// Older kernels do not identify anonymous VFIO device FDs in
		// fdinfo. Treat an unidentified one as busy rather than risk an
		// unbind that waits indefinitely in the VFIO core.
		if matches || !identified {
			return true, nil
		}
	}

	fdInfo, err := os.Stat(fdPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("inspect process %s file descriptor %s: %w", pid, fd, err)
	}
	for _, identity := range identities {
		if os.SameFile(identity.info, fdInfo) {
			return true, nil
		}
	}
	return false, nil
}

func (o *hostPCIDriverOps) anonymousVFIODeviceMatches(pid, fd, pciAddr string) (bool, bool, error) {
	fdInfoPath := filepath.Join(o.procPath, pid, "fdinfo", fd)
	data, err := os.ReadFile(fdInfoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("inspect process %s file descriptor %s metadata: %w", pid, fd, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		syspath, found := strings.CutPrefix(line, "vfio-device-syspath:")
		if !found {
			continue
		}
		for _, component := range strings.Split(filepath.Clean(strings.TrimSpace(syspath)), string(filepath.Separator)) {
			if component == pciAddr {
				return true, true, nil
			}
		}
		return false, true, nil
	}
	return false, false, nil
}

func (o *hostPCIDriverOps) vfioDeviceNodes(pciAddr string) ([]string, error) {
	devicePath := filepath.Join(o.pciBusPath, "devices", pciAddr)
	groupLink, err := os.Readlink(filepath.Join(devicePath, "iommu_group"))
	if err != nil {
		return nil, fmt.Errorf("resolve IOMMU group: %w", err)
	}
	groupNode := filepath.Join(o.vfioDevPath, filepath.Base(groupLink))
	nodes := []string{groupNode}

	cdevEntries, err := filepath.Glob(filepath.Join(devicePath, "vfio-dev", "vfio*"))
	if err != nil {
		return nil, err
	}
	for _, entry := range cdevEntries {
		nodes = append(nodes, filepath.Join(o.vfioDevPath, "devices", filepath.Base(entry)))
	}

	existing := nodes[:0]
	for _, node := range nodes {
		if _, err := os.Stat(node); err == nil {
			existing = append(existing, node)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	if len(existing) == 0 {
		return nil, fmt.Errorf("no VFIO device node found for IOMMU group %s", filepath.Base(groupLink))
	}
	sort.Strings(existing)
	return existing, nil
}

func (o *hostPCIDriverOps) processDescription(pid string) string {
	comm, err := os.ReadFile(filepath.Join(o.procPath, pid, "comm"))
	if err != nil {
		return fmt.Sprintf("process %s", pid)
	}
	return fmt.Sprintf("process %s (%s)", pid, strings.TrimSpace(string(comm)))
}

func (o *hostPCIDriverOps) SetDriverOverride(pciAddr, driver string) error {
	value := driver
	if driver == "" {
		value = "\n"
	}
	return os.WriteFile(filepath.Join(o.pciBusPath, "devices", pciAddr, "driver_override"), []byte(value), 0)
}

func (o *hostPCIDriverOps) Unbind(pciAddr, driver string) error {
	return os.WriteFile(filepath.Join(o.pciBusPath, "drivers", driver, "unbind"), []byte(pciAddr), 0)
}

func (o *hostPCIDriverOps) Bind(pciAddr, driver string) error {
	return os.WriteFile(filepath.Join(o.pciBusPath, "drivers", driver, "bind"), []byte(pciAddr), 0)
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
