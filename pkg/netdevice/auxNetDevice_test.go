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

package netdevice_test

import (
	"github.com/jaypipes/ghw"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AuxNetDevice", func() {
	Describe("creation", func() {
		Context("correct input", func() {
			It("should create 2 devices", func() {
				rc := &types.ResourceConfig{}
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/foo.bar.0",
						"sys/bus/pci/devices/0000:00:00.1/foo.bar.1",
						"sys/bus/pci/drivers/foo",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/foo",
					},
				}
				defer fs.Use()()
				f := factory.NewResourceFactory("fake", "fake", true)
				dev := &ghw.PCIDevice{Address: "0000:00:00.1"}

				auxDevs, err := netdevice.NewAuxNetDevices(dev, f, rc)
				Expect(auxDevs).ToNot(BeNil())
				Expect(auxDevs).To(HaveLen(len(fs.Dirs) - 1))
				Expect(err).ToNot(HaveOccurred())
			})
			It("should create device with correct field values", func() {
				rc := &types.ResourceConfig{}
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/auxiliary/devices",
						"sys/bus/pci/devices/0000:00:00.1/foo.bar.0/net/eth0",
						"sys/bus/pci/devices/0000:00:00.1/net/net0",
						"sys/bus/pci/drivers/foo",
					},
					Symlinks: map[string]string{
						"sys/bus/auxiliary/devices/foo.bar.0":     "../../../bus/pci/devices/0000:00:00.1/foo.bar.0",
						"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/foo",
					},
				}
				defer fs.Use()()
				f := factory.NewResourceFactory("fake", "fake", true)
				dev := &ghw.PCIDevice{Address: "0000:00:00.1"}

				fakeSriovnetProvider := mocks.SriovnetProvider{}
				fakeSriovnetProvider.
					On("GetUplinkRepresentorFromAux", mock.AnythingOfType("string")).Return("net0", nil)
				utils.SetSriovnetProviderInst(&fakeSriovnetProvider)

				auxDevs, err := netdevice.NewAuxNetDevices(dev, f, rc)
				Expect(auxDevs).ToNot(BeNil())
				Expect(auxDevs).To(HaveLen(1))
				Expect(err).ToNot(HaveOccurred())
				auxDev, _ := auxDevs[0].(types.PciNetDevice)
				Expect(auxDev.GetDriver()).To(Equal("foo"))
				Expect(auxDev.GetNetName()).To(Equal("eth0"))
				Expect(auxDev.GetEnvVal()).To(Equal("foo.bar.0"))
				Expect(auxDev.GetDeviceSpecs()).To(HaveLen(0))
				Expect(auxDev.GetLinkSpeed()).To(Equal(""))
				Expect(auxDev.GetPFName()).To(Equal("net0"))
				Expect(auxDev.GetRdmaSpec()).To(BeNil())
				Expect(auxDev.GetLinkType()).To(Equal("fakeLinkType"))
				Expect(auxDev.GetAPIDevice().ID).To(Equal("foo.bar.0"))
			})
		})
		Context("incorrect input", func() {
			It("invalid pci device", func() {
				rc := &types.ResourceConfig{}
				f := factory.NewResourceFactory("fake", "fake", true)
				dev := &ghw.PCIDevice{Address: "0000:00:00.1"}

				auxDevs, err := netdevice.NewAuxNetDevices(dev, f, rc)
				Expect(auxDevs).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
			It("no ifName", func() {
				rc := &types.ResourceConfig{}
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/foo.bar.0",
						"sys/bus/pci/drivers/foo",
					},
					Symlinks: map[string]string{"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/foo"},
				}
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()
				f := factory.NewResourceFactory("fake", "fake", true)
				dev := &ghw.PCIDevice{Address: "0000:00:00.1"}

				auxDevs, err := netdevice.NewAuxNetDevices(dev, f, rc)
				Expect(auxDevs).ToNot(BeNil())
				Expect(auxDevs).To(HaveLen(1))
				Expect(err).ToNot(HaveOccurred())
				auxDev, _ := auxDevs[0].(types.PciNetDevice)
				Expect(auxDev.GetNetName()).To(Equal(""))
				Expect(auxDev.GetLinkType()).To(Equal(""))
			})
			It("no auxiliary devices", func() {
				rc := &types.ResourceConfig{}
				fs := &utils.FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.1"}}
				defer fs.Use()()
				f := factory.NewResourceFactory("fake", "fake", true)
				dev := &ghw.PCIDevice{Address: "0000:00:00.1"}

				auxDevs, err := netdevice.NewAuxNetDevices(dev, f, rc)
				Expect(auxDevs).ToNot(BeNil())
				Expect(auxDevs).To(HaveLen(0))
				Expect(err).ToNot(HaveOccurred())
			})
			It("no driver name", func() {
				rc := &types.ResourceConfig{}
				fs := &utils.FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.1/foo.bar.0"}}
				defer fs.Use()()
				f := factory.NewResourceFactory("fake", "fake", true)
				dev := &ghw.PCIDevice{Address: "0000:00:00.1"}

				auxDevs, err := netdevice.NewAuxNetDevices(dev, f, rc)
				Expect(auxDevs).ToNot(BeNil())
				Expect(auxDevs).To(HaveLen(0))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
