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

package auxnetdevice_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/auxnetdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	tmocks "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"
)

var _ = Describe("AuxNetDeviceProvider", func() {
	DescribeTable("validating configuration",
		func(rc *types.ResourceConfig, expected bool) {
			rf := factory.NewResourceFactory("fake", "fake", true)
			p := auxnetdevice.NewAuxNetDeviceProvider(rf)
			actual := p.ValidConfig(rc)
			Expect(actual).To(Equal(expected))
		},
		Entry("invalid selector in config passed",
			&types.ResourceConfig{SelectorObj: &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{}}},
			false),
		Entry("auxTypes list is empty",
			&types.ResourceConfig{SelectorObj: &types.AuxNetDeviceSelectors{AuxTypes: []string{}}},
			false),
		Entry("unsupported auxiliary device types specified",
			&types.ResourceConfig{SelectorObj: &types.AuxNetDeviceSelectors{AuxTypes: []string{"sf", "eth", "rdma"}}},
			false),
		Entry("supported auxiliary device types",
			&types.ResourceConfig{SelectorObj: &types.AuxNetDeviceSelectors{AuxTypes: []string{"sf", "sf"}}},
			true),
	)
	Describe("getting new instance of auxNetDeviceProvider", func() {
		Context("with correct arguments", func() {
			It("should return valid instance of the provider", func() {
				rf := &tmocks.ResourceFactory{}
				p := auxnetdevice.NewAuxNetDeviceProvider(rf)
				Expect(p).ToNot(BeNil())
			})
		})
	})
	Describe("getting devices", func() {
		Context("when there are none", func() {
			rf := &tmocks.ResourceFactory{}
			p := auxnetdevice.NewAuxNetDeviceProvider(rf)
			config := &types.ResourceConfig{}
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config)
			It("should return empty slice", func() {
				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})
		})
	})
	Describe("adding 3 target devices", func() {
		Context("when 2 are valid devices, but only one have auxiliary devices", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/auxiliary/devices",
					"sys/bus/pci/devices/0000:01:00.0",
					"sys/bus/pci/devices/0000:02:00.0",
					"sys/bus/pci/devices/0000:03:00.0",
					"sys/bus/pci/drivers/mlx5_core",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:01:00.0/driver": "../../../../bus/pci/drivers/mlx5_core",
					"sys/bus/pci/devices/0000:02:00.0/driver": "../../../../bus/pci/drivers/mlx5_core",
					"sys/bus/pci/devices/0000:03:00.0/driver": "../../../../bus/pci/drivers/mlx5_core",
				},
			}

			defer fs.Use()()
			utils.SetDefaultMockNetlinkProvider()

			fakeSriovnetProvider := mocks.SriovnetProvider{}
			fakeSriovnetProvider.
				On("GetUplinkRepresentorFromAux", mock.AnythingOfType("string")).Return("ens1f0", nil).
				On("GetPfPciFromAux", mock.AnythingOfType("string")).Return("0000:01:00.0", nil).
				On("GetSfIndexByAuxDev", mock.AnythingOfType("string")).Return(0, nil).
				On("GetNetDevicesFromAux", mock.AnythingOfType("string")).Return([]string{""}, nil).
				On("GetAuxNetDevicesFromPci", "0000:01:00.0").Return([]string{"mlx5_core.sf.1", "mlx5_core.sf.2", "mlx5_core.sf.3"}, nil).
				On("GetAuxNetDevicesFromPci", "0000:02:00.0").Return([]string{}, nil)
			utils.SetSriovnetProviderInst(&fakeSriovnetProvider)

			rf := factory.NewResourceFactory("fake", "fake", true)
			p := auxnetdevice.NewAuxNetDeviceProvider(rf)
			config := &types.ResourceConfig{
				DeviceType: types.AuxNetDeviceType,
				SelectorObj: &types.AuxNetDeviceSelectors{
					DeviceSelectors: types.DeviceSelectors{},
				},
			}

			vendor := &pcidb.Vendor{Name: "Mellanox Technologies"}
			devsToAdd := []*ghw.PCIDevice{
				{
					Address: "0000:01:00.0",
					Class:   &pcidb.Class{ID: "2"},
					Vendor:  vendor,
					Product: &pcidb.Product{Name: "MT42822 BlueField-2 integrated ConnectX-6 Dx network controller"},
				},
				{
					Address: "0000:02:00.0",
					Class:   &pcidb.Class{ID: "2"},
					Vendor:  vendor,
					Product: &pcidb.Product{Name: "MT2892 Family [ConnectX-6 Dx]"},
				},
				{
					Address: "0000:03:00.0",
					Class:   &pcidb.Class{ID: "7"},
					Vendor:  vendor,
					Product: &pcidb.Product{Name: "MT28908 Family [ConnectX-6]"},
				},
			}

			err := p.AddTargetDevices(devsToAdd, 0x2)
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config)
			It("shouldn't return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return only 1 device on GetDiscoveredDevices()", func() {
				Expect(dDevs).To(HaveLen(2))
			})
			It("should return 3 auxiliary devices on GetDevices()", func() {
				Expect(devs).To(HaveLen(3))
			})
		})
	})
	Describe("getting Filtered devices", func() {
		Context("using selectors", func() {
			It("should correctly filter devices", func() {
				rf := factory.NewResourceFactory("fake", "fake", false)
				p := auxnetdevice.NewAuxNetDeviceProvider(rf)
				all := make([]types.HostDevice, 5)
				mocked := make([]tmocks.AuxNetDevice, 5)

				ve := []string{"8086", "15b3", "15b3", "15b3", "8086"}
				de := []string{"1521", "101b", "a2d6", "a2d6", "1521"}
				md := []string{"igb", "mlx5_core", "mlx5_core", "mlx5_core", "igb"}
				pf := []string{"eth0", "ib0", "net0", "net0", "eth1"}
				ro := []string{"0000:86:00.0", "0000:03:00.0", "0000:04:00.0", "0000:04:00.0", "0000:86:00.4"}
				lt := []string{"ether", "infiniband", "ether", "ether", "ether"}
				rd := []bool{false, true, false, false, true}
				at := []string{"eth", "rdma", "sf", "sf", "eth"}

				for i := range mocked {
					mocked[i].
						On("GetVendor").Return(ve[i]).
						On("GetDeviceCode").Return(de[i]).
						On("GetDriver").Return(md[i]).
						On("GetPfNetName").Return(pf[i]).
						On("GetPfPciAddr").Return(ro[i]).
						On("GetLinkType").Return(lt[i]).
						On("GetFuncID").Return(-1).
						On("IsRdma").Return(rd[i]).
						On("GetAuxType").Return(at[i])

					all[i] = &mocked[i]
				}

				testCases := []struct {
					name     string
					sel      *types.AuxNetDeviceSelectors
					expected []types.HostDevice
				}{
					{"vendors", &types.AuxNetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Vendors: []string{"8086"}}}, []types.HostDevice{all[0], all[4]}},
					{"devices", &types.AuxNetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Devices: []string{"a2d6"}}}, []types.HostDevice{all[2], all[3]}},
					{"drivers", &types.AuxNetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Drivers: []string{"mlx5_core"}}}, []types.HostDevice{all[1], all[2], all[3]}},
					{"pfNames", &types.AuxNetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{PfNames: []string{"net0", "eth1"}}}, []types.HostDevice{all[2], all[3], all[4]}},
					{"rootDevices", &types.AuxNetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{RootDevices: []string{"0000:86:00.0", "0000:86:00.4"}}}, []types.HostDevice{all[0], all[4]}},
					{"linkTypes", &types.AuxNetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{LinkTypes: []string{"infiniband"}}}, []types.HostDevice{all[1]}},
					{"linkTypes multi", &types.AuxNetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{LinkTypes: []string{"infiniband", "ether"}}}, all},
					{"rdma", &types.AuxNetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{IsRdma: true}}, []types.HostDevice{all[1], all[4]}},
					{"auxTypes", &types.AuxNetDeviceSelectors{AuxTypes: []string{"sf", "sf"}}, []types.HostDevice{all[2], all[3]}},
				}

				for _, tc := range testCases {
					By(tc.name)
					config := &types.ResourceConfig{SelectorObj: tc.sel}
					actual, err := p.GetFilteredDevices(all, config)
					Expect(err).NotTo(HaveOccurred())
					Expect(actual).To(HaveLen(len(tc.expected)))
					Expect(actual).To(ConsistOf(tc.expected))
				}
			})
		})
	})
})
