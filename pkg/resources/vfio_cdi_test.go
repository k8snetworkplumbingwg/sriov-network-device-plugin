// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"errors"
	"testing"

	cdilib "github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

func TestNativeCDIRecovery(t *testing.T) {
	registry := cdilib.GetRegistry()
	oldDirs := registry.GetSpecDirectories()
	if err := registry.Configure(cdilib.WithSpecDirs(t.TempDir()), cdilib.WithAutoRefresh(false)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := registry.Configure(cdilib.WithSpecDirs(oldDirs...), cdilib.WithAutoRefresh(true)); err != nil {
			t.Error(err)
		}
	})
	good := &allocationDevice{id: "0000:01:00.1", specs: []*pluginapi.DeviceSpec{
		{HostPath: "/dev/null", ContainerPath: "/dev/vfio/devices/vfio1", Permissions: "rw"},
	}}
	repaired := &allocationDevice{id: "0000:01:00.2", err: errors.New("missing cdev"), specs: []*pluginapi.DeviceSpec{
		{HostPath: "/dev/null", ContainerPath: "/dev/vfio/devices/vfio2", Permissions: "rw"},
		{HostPath: "/dev/zero", ContainerPath: "/dev/iommu", Permissions: "rw"},
	}}
	pool := NewResourcePool(&types.ResourceConfig{ResourceName: "native", VFIOMode: types.VFIOModeIOMMUFD},
		map[string]types.HostDevice{good.id: good, repaired.id: repaired})
	resource, err := NewResourceServer("example.com", "test", false, true, pool)
	if err != nil {
		t.Fatal(err)
	}
	server := resource.(*resourceServer)
	// A missing device must not stop ListAndWatch from publishing the healthy one.
	if err := server.updateCDISpec(); err != nil {
		t.Fatal(err)
	}
	checkCDIDevice(t, registry, good.id, 1)
	if registry.DeviceDB().GetDevice("example.com/net-pci="+repaired.id) != nil {
		t.Fatal("unavailable native device was published")
	}
	// Repair before the next health probe. Allocation must refresh the CDI file.
	repaired.err = nil
	request := &pluginapi.AllocateRequest{ContainerRequests: []*pluginapi.ContainerAllocateRequest{{DevicesIds: []string{repaired.id}}}}
	response, err := server.Allocate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.ContainerResponses[0].Devices) != 0 {
		t.Fatal("CDI allocation also returned direct device nodes")
	}
	_, ids, err := cdilib.ParseAnnotations(response.ContainerResponses[0].Annotations)
	if err != nil || len(ids) != 1 || ids[0] != "example.com/net-pci="+repaired.id {
		t.Fatalf("wrong CDI annotation: %v, %v", ids, err)
	}
	checkCDIDevice(t, registry, repaired.id, 2)
	// Another refresh must retain a repaired device even while cached health is stale.
	if err := server.updateCDISpec(); err != nil {
		t.Fatal(err)
	}
	checkCDIDevice(t, registry, repaired.id, 2)
	repaired.err = errors.New("cdev disappeared")
	if response, err := server.Allocate(context.Background(), request); err == nil || response != nil {
		t.Fatalf("unavailable native device allocated using stale CDI entry: %v, %v", response, err)
	}
}

func checkCDIDevice(t *testing.T, registry cdilib.Registry, id string, count int) {
	t.Helper()
	if err := registry.Refresh(); err != nil {
		t.Fatal(err)
	}
	device := registry.DeviceDB().GetDevice("example.com/net-pci=" + id)
	if device == nil {
		t.Fatalf("missing CDI device %s", id)
	}
	if len(device.ContainerEdits.DeviceNodes) != count {
		t.Fatalf("unexpected CDI nodes: %v", device.ContainerEdits.DeviceNodes)
	}
	for _, node := range device.ContainerEdits.DeviceNodes {
		if node.Path != "/dev/iommu" && node.Path != "/dev/vfio/devices/vfio1" && node.Path != "/dev/vfio/devices/vfio2" {
			t.Fatalf("unexpected native CDI path: %s", node.Path)
		}
	}
}
