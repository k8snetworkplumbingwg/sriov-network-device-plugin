/*
 * SPDX-FileCopyrightText: Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package devices_test

import (
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

var _ = Describe("HostDevice", func() {
	t := GinkgoT()
	newPciDeviceFn := func(pciAddr string) *ghw.PCIDevice {
		return &ghw.PCIDevice{
			Address: pciAddr,
			Vendor:  &pcidb.Vendor{ID: "Vendor"},
			Product: &pcidb.Product{ID: "Product"},
		}
	}
	Context("Create new HostDevice", func() {
		It("should fail if cannot get driver name", func() {
			fs := &utils.FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/driver": []byte("not a symlink")},
			}
			defer fs.Use()()

			f := factory.NewResourceFactory("fake", "fake", true, false)
			pciAddr := "0000:00:00.1"
			in := newPciDeviceFn(pciAddr)
			rc := &types.ResourceConfig{}
			infoProviders := make([]types.DeviceInfoProvider, 0)

			dev, err := devices.NewHostDeviceImpl(in, pciAddr, f, rc, infoProviders)

			Expect(dev).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
		It("should populate fields", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1/net/eth0",
					"sys/bus/pci/drivers/fake",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/fake",
				},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0")},
			}
			defer fs.Use()()

			f := factory.NewResourceFactory("fake", "fake", true, false)
			pciAddr := "0000:00:00.1"
			in := newPciDeviceFn(pciAddr)
			rc := &types.ResourceConfig{}
			infoProviders := make([]types.DeviceInfoProvider, 0)

			dev, err := devices.NewHostDeviceImpl(in, pciAddr, f, rc, infoProviders)

			Expect(err).NotTo(HaveOccurred())
			Expect(dev.GetVendor()).To(Equal("Vendor"))
			Expect(dev.GetDeviceCode()).To(Equal("Product"))
			Expect(dev.GetDriver()).To(Equal("fake"))
			Expect(dev.GetDeviceID()).To(Equal(pciAddr))
			Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
		})
		It("should not populate topology due to negative numa_node", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1/net/eth0",
					"sys/bus/pci/drivers/vfio-pci",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/vfio-pci",
				},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("-1")},
			}
			defer fs.Use()()

			f := factory.NewResourceFactory("fake", "fake", true, false)
			pciAddr := "0000:00:00.1"
			in := newPciDeviceFn(pciAddr)
			rc := &types.ResourceConfig{}
			infoProviders := make([]types.DeviceInfoProvider, 0)

			dev, err := devices.NewHostDeviceImpl(in, pciAddr, f, rc, infoProviders)

			Expect(dev.GetAPIDevice().Topology).To(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})
		It("should not populate topology due to missing numa_node", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1/net/eth0",
					"sys/bus/pci/drivers/vfio-pci",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/vfio-pci",
				},
			}
			defer fs.Use()()

			f := factory.NewResourceFactory("fake", "fake", true, false)
			pciAddr := "0000:00:00.1"
			in := newPciDeviceFn(pciAddr)
			rc := &types.ResourceConfig{}
			infoProviders := make([]types.DeviceInfoProvider, 0)

			dev, err := devices.NewHostDeviceImpl(in, pciAddr, f, rc, infoProviders)

			Expect(dev.GetAPIDevice().Topology).To(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})
		It("should not populate topology due to config option being set", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1/net/eth0",
					"sys/bus/pci/drivers/vfio-pci",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/vfio-pci",
				},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0")},
			}
			defer fs.Use()()

			f := factory.NewResourceFactory("fake", "fake", true, false)
			pciAddr := "0000:00:00.1"
			in := newPciDeviceFn(pciAddr)
			rc := &types.ResourceConfig{ExcludeTopology: true}
			infoProviders := make([]types.DeviceInfoProvider, 0)

			dev, err := devices.NewHostDeviceImpl(in, pciAddr, f, rc, infoProviders)

			Expect(dev.GetAPIDevice().Topology).To(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Context("Create new device with info providers", func() {
		rc := &types.ResourceConfig{
			ResourceName:   "fake",
			ResourcePrefix: "fake",
		}
		fs := &utils.FakeFilesystem{
			Dirs: []string{
				"sys/bus/pci/devices/0000:00:00.1",
				"sys/bus/pci/devices/0000:00:00.2/net/eth1",
				"sys/bus/pci/drivers/mlx5_core",
			},
			Symlinks: map[string]string{
				"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/mlx5_core",
				"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/mlx5_core",
			},
			Files: map[string][]byte{
				"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("1"),
				"sys/bus/pci/devices/0000:00:00.2/numa_node": []byte("2"),
			},
		}

		f := &mocks.ResourceFactory{}

		pciAddr1 := "0000:00:00.1"
		pciAddr2 := "0000:00:00.2"
		mockInfo1 := &mocks.DeviceInfoProvider{}
		mockSpec1 := []*v1beta1.DeviceSpec{
			{HostPath: "/mock/spec/1"},
		}
		mockEnv1 := types.AdditionalInfo{"deviceID": pciAddr1}
		mockInfo1.On("GetName").Return("generic")
		mockInfo1.On("GetEnvVal").Return(mockEnv1)
		mockInfo1.On("GetDeviceSpecs").Return(mockSpec1)
		mockInfo1.On("GetMounts").Return(nil)
		mockInfo2 := &mocks.DeviceInfoProvider{}
		mockSpec2 := []*v1beta1.DeviceSpec{
			{HostPath: "/mock/spec/2"},
		}
		mockEnv2 := types.AdditionalInfo{"deviceID": pciAddr2}
		mockInfo2.On("GetName").Return("generic")
		mockInfo2.On("GetEnvVal").Return(mockEnv2)
		mockInfo2.On("GetDeviceSpecs").Return(mockSpec2)
		mockInfo2.On("GetMounts").Return(nil)
		f.On("GetDefaultInfoProvider", pciAddr1, "mlx5_core").Return([]types.DeviceInfoProvider{mockInfo1}).
			On("GetDefaultInfoProvider", pciAddr2, "mlx5_core").Return([]types.DeviceInfoProvider{mockInfo2})

		in1 := newPciDeviceFn(pciAddr1)
		in2 := newPciDeviceFn(pciAddr2)

		It("should populate infoProviders if zero were passed", func() {
			defer fs.Use()()
			utils.SetDefaultMockNetlinkProvider()
			infoProviders := make([]types.DeviceInfoProvider, 0)
			dev, err := devices.NewHostDeviceImpl(in1, pciAddr1, f, rc, infoProviders)

			Expect(dev.GetDriver()).To(Equal("mlx5_core"))
			envs := dev.GetEnvVal()
			Expect(len(envs)).To(Equal(1))
			_, exist := envs["generic"]
			Expect(exist).To(BeTrue())
			pci, exist := envs["generic"]["deviceID"]
			Expect(exist).To(BeTrue())
			Expect(pci).To(Equal(pciAddr1))
			Expect(dev.GetDeviceSpecs()).To(Equal(mockSpec1))
			Expect(dev.GetMounts()).To(HaveLen(0))
			Expect(err).NotTo(HaveOccurred())
			mockInfo1.AssertExpectations(t)
		})
		It("should not populate infoProviders if some were passed", func() {
			defer fs.Use()()
			utils.SetDefaultMockNetlinkProvider()
			infoProviders := []types.DeviceInfoProvider{mockInfo2}
			dev, err := devices.NewHostDeviceImpl(in2, pciAddr2, f, rc, infoProviders)

			Expect(dev.GetDriver()).To(Equal("mlx5_core"))
			envs := dev.GetEnvVal()
			Expect(len(envs)).To(Equal(1))
			_, exist := envs["generic"]
			Expect(exist).To(BeTrue())
			pci, exist := envs["generic"]["deviceID"]
			Expect(exist).To(BeTrue())
			Expect(pci).To(Equal(pciAddr2))
			Expect(dev.GetDeviceSpecs()).To(Equal(mockSpec2))
			Expect(dev.GetMounts()).To(HaveLen(0))
			Expect(err).NotTo(HaveOccurred())
			mockInfo1.AssertExpectations(t)
		})
	})
})
