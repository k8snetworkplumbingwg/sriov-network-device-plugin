// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package infoprovider_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	"golang.org/x/sys/unix"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

func nativeVFIOFixture(t *testing.T) *utils.FakeFilesystem {
	t.Helper()
	var st unix.Stat_t
	if err := unix.Stat("/dev/null", &st); err != nil {
		t.Fatal(err)
	}
	fs := utils.FakeFilesystem{
		Dirs:  []string{"sys/bus/pci/devices/0000:af:02.0/vfio-dev/vfio2", "sys/kernel/iommu_groups/172", "host-dev/vfio/devices", "sys/class/misc/iommu"},
		Files: map[string][]byte{"sys/bus/pci/devices/0000:af:02.0/vfio-dev/vfio2/dev": []byte(fmt.Sprintf("%d:%d", unix.Major(st.Rdev), unix.Minor(st.Rdev)))},
		Symlinks: map[string]string{
			"sys/bus/pci/devices/0000:af:02.0/driver":      "/sys/bus/pci/drivers/vfio-pci",
			"sys/bus/pci/devices/0000:af:02.0/iommu_group": "../../../../kernel/iommu_groups/172",
			"host-dev/vfio/devices/vfio2":                  "/dev/null",
		},
	}
	t.Cleanup(fs.Use())
	if err := os.WriteFile(filepath.Join(fs.RootDir, "sys/class/misc/iommu/dev"), []byte(fmt.Sprintf("%d:%d", unix.Major(st.Rdev), unix.Minor(st.Rdev))), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/null", filepath.Join(fs.RootDir, "host-dev/iommu")); err != nil {
		t.Fatal(err)
	}

	return &fs
}

func TestNativeVFIOSelection(t *testing.T) {
	fs := nativeVFIOFixture(t)
	p := infoprovider.NewVfioInfoProviderWithConfig("0000:af:02.0", &types.ResourceConfig{VFIOMode: "iommufd", HostDevDir: filepath.Join(fs.RootDir, "host-dev")})
	// Call metadata first to detect getter-order dependencies.
	env := p.GetEnvVal()
	if env["cdev-mount"] != "/dev/vfio/devices/vfio2" || env["mount"] != "" || env["dev-mount"] != "" {
		t.Fatalf("wrong native metadata: %v", env)
	}
	specs, err := p.GetDeviceSpecs()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || specs[0].HostPath != "/dev/vfio/devices/vfio2" || specs[0].ContainerPath != specs[0].HostPath {
		b, _ := json.Marshal(specs)
		t.Fatalf("wrong native device specs: %s", b)
	}
	// Exercise configuration through the real factory and host-device constructor.
	rc := &types.ResourceConfig{VFIOMode: "iommufd", HostDevDir: filepath.Join(fs.RootDir, "host-dev"), ExcludeTopology: true}
	pci := &ghw.PCIDevice{Address: "0000:af:02.0", Vendor: &pcidb.Vendor{ID: "15b3"}, Product: &pcidb.Product{ID: "101e"}}
	host, err := devices.NewHostDeviceImpl(pci, pci.Address, factory.NewResourceFactory("example.com", "test", false), rc, nil)
	if err != nil {
		t.Fatal(err)
	}
	fromHost, err := host.GetDeviceSpecs()
	if err != nil || len(fromHost) != 1 || fromHost[0].HostPath != specs[0].HostPath {
		t.Fatalf("native configuration lost in factory: %v, %v", fromHost, err)
	}
	checkNativeRemap(t, p, fs.RootDir)
	checkNativeControl(t, p, fs.RootDir)
}

func checkNativeRemap(t *testing.T, p types.DeviceInfoProvider, root string) {
	t.Helper()
	// Rebinding can renumber a cdev. Do not mix old metadata with a new node.
	oldSys := filepath.Join(root, "sys/bus/pci/devices/0000:af:02.0/vfio-dev/vfio2")
	newSys := filepath.Join(root, "sys/bus/pci/devices/0000:af:02.0/vfio-dev/vfio3")
	oldNode := filepath.Join(root, "host-dev/vfio/devices/vfio2")
	newNode := filepath.Join(root, "host-dev/vfio/devices/vfio3")
	if err := os.Rename(oldSys, newSys); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldNode, newNode); err != nil {
		t.Fatal(err)
	}
	if partial, err := p.GetDeviceSpecs(); err == nil || len(partial) != 0 {
		t.Fatalf("cdev remapping accepted: %v, %v", partial, err)
	}
	if p.GetEnvVal()["cdev-mount"] != "/dev/vfio/devices/vfio2" {
		t.Fatal("metadata changed after initial resolution")
	}
	if err := os.Rename(newSys, oldSys); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(newNode, oldNode); err != nil {
		t.Fatal(err)
	}
}

func checkNativeControl(t *testing.T, p types.DeviceInfoProvider, root string) {
	t.Helper()
	withIommu := infoprovider.NewVfioInfoProviderWithConfig("0000:af:02.0", &types.ResourceConfig{VFIOMode: "iommufd", NeedIommu: true, HostDevDir: filepath.Join(root, "host-dev")})
	withSpecs, err := withIommu.GetDeviceSpecs()
	if err != nil || len(withSpecs) != 2 || withSpecs[1].HostPath != "/dev/iommu" {
		t.Fatalf("control injection: %v, %v", withSpecs, err)
	}
	if withIommu.GetEnvVal()["iommu-mount"] != "/dev/iommu" {
		t.Fatal("missing control metadata")
	}
	if err := os.Remove(filepath.Join(root, "host-dev/iommu")); err != nil {
		t.Fatal(err)
	}
	if partial, err := withIommu.GetDeviceSpecs(); err == nil || len(partial) != 0 {
		t.Fatalf("missing control node accepted: %v, %v", partial, err)
	}

	if err := os.Remove(filepath.Join(root, "host-dev/vfio/devices/vfio2")); err != nil {
		t.Fatal(err)
	}

	if partial, err := p.GetDeviceSpecs(); err == nil || len(partial) != 0 {
		t.Fatalf("missing cdev accepted or fell back: %v, %v", partial, err)
	}
}
