// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"errors"
	"strings"
	"testing"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

// The fixture replaces hardware discovery. Allocation and aggregation remain real.
type allocationDevice struct {
	types.HostDevice
	provider types.DeviceInfoProvider
	id       string
	specs    []*pluginapi.DeviceSpec
	err      error
}

func (d *allocationDevice) GetAPIDevice() *pluginapi.Device {
	return &pluginapi.Device{ID: d.id, Health: pluginapi.Healthy}
}
func (d *allocationDevice) GetDeviceSpecs() ([]*pluginapi.DeviceSpec, error) {
	if d.provider != nil {
		return d.provider.GetDeviceSpecs()
	}
	return d.specs, d.err
}
func (d *allocationDevice) GetEnvVal() map[string]types.AdditionalInfo {
	if d.provider != nil {
		return map[string]types.AdditionalInfo{d.provider.GetName(): d.provider.GetEnvVal()}
	}
	return map[string]types.AdditionalInfo{}
}
func (d *allocationDevice) GetMounts() []*pluginapi.Mount { return nil }

func TestVFIOAllocation(t *testing.T) {
	good := &allocationDevice{id: "0000:01:00.1", specs: []*pluginapi.DeviceSpec{{HostPath: "/dev/vfio/devices/vfio1", ContainerPath: "/dev/vfio/devices/vfio1", Permissions: "rw"}, {HostPath: "/dev/iommu", ContainerPath: "/dev/iommu", Permissions: "rw"}}}
	other := &allocationDevice{id: "0000:01:00.2", specs: []*pluginapi.DeviceSpec{{HostPath: "/dev/vfio/devices/vfio2", ContainerPath: "/dev/vfio/devices/vfio2", Permissions: "rw"}, {HostPath: "/dev/iommu", ContainerPath: "/dev/iommu", Permissions: "rw"}}}
	bad := &allocationDevice{id: "0000:01:00.3", err: errors.New("missing cdev")}
	pool := NewResourcePool(&types.ResourceConfig{ResourceName: "native", VFIOMode: types.VFIOModeIOMMUFD}, map[string]types.HostDevice{good.id: good, other.id: other, bad.id: bad})
	server := NewResourceServer("example.com", "test", false, pool)
	for _, tc := range []struct {
		name    string
		ids     []string
		invalid bool
	}{
		{"two native devices", []string{good.id, other.id}, false},
		{"partial request", []string{good.id, bad.id}, true},
		{"unknown allocation", []string{good.id, "unknown"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := server.Allocate(context.Background(), &pluginapi.AllocateRequest{ContainerRequests: []*pluginapi.ContainerAllocateRequest{{DevicesIDs: tc.ids}}})
			if tc.invalid {
				if err == nil || response != nil {
					t.Fatalf("partial allocation succeeded: response=%v error=%v", response, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(response.ContainerResponses) != 1 || len(response.ContainerResponses[0].Devices) != 3 {
				t.Fatalf("control node not deduplicated: %v", response)
			}
		})
	}
}

func TestNativeVFIOHealth(t *testing.T) {
	device := &allocationDevice{id: "0000:01:00.1", err: errors.New("missing cdev")}
	pool := NewResourcePool(&types.ResourceConfig{VFIOMode: types.VFIOModeIOMMUFD}, map[string]types.HostDevice{device.id: device})
	if pool.GetDevices()[device.id].Health != pluginapi.Unhealthy {
		t.Fatal("missing cdev advertised healthy")
	}
	device.err = nil
	if !pool.Probe() || pool.GetDevices()[device.id].Health != pluginapi.Healthy {
		t.Fatal("repaired cdev health did not recover")
	}
	device.err = errors.New("cdev removed")
	if !pool.Probe() || pool.GetDevices()[device.id].Health != pluginapi.Unhealthy {
		t.Fatal("removed cdev remained healthy")
	}
}

func TestFirstUIOAllocationMetadata(t *testing.T) {
	fs := utils.FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:02:00.0/uio/uio0"}}
	defer fs.Use()()
	device := &allocationDevice{id: "0000:02:00.0", provider: infoprovider.NewUioInfoProvider("0000:02:00.0")}
	pool := NewResourcePool(&types.ResourceConfig{ResourceName: "uio"}, map[string]types.HostDevice{device.id: device})
	server := NewResourceServer("example.com", "test", false, pool)
	response, err := server.Allocate(context.Background(), &pluginapi.AllocateRequest{ContainerRequests: []*pluginapi.ContainerAllocateRequest{{DevicesIDs: []string{device.id}}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.ContainerResponses[0].Envs["PCIDEVICE_EXAMPLE_COM_UIO_INFO"], `"mount":"/dev/uio0"`) {
		t.Fatalf("first allocation lost UIO metadata: %v", response)
	}
}
