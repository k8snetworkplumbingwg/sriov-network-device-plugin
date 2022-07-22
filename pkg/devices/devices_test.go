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

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

var _ = Describe("Devices", func() {
	Describe("PciDevice", func() {
		Context("Create new PciDevice", func() {
			It("should populate fields", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.0",
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/fake",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/fake",
						"sys/bus/pci/devices/0000:00:00.1/physfn":      "../0000:00:00.0",
						"sys/bus/pci/devices/0000:00:00.0/virtfn0":     "../0000:00:00.1",
					},
				}
				defer fs.Use()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{
					Address: "0000:00:00.1",
					Vendor:  &pcidb.Vendor{},
					Product: &pcidb.Product{},
				}
				rc := &types.ResourceConfig{}
				infoProviders := make([]types.DeviceInfoProvider, 0)

				dev, err := devices.NewPciDevice(in, f, rc, infoProviders)

				Expect(dev.GetDriver()).To(Equal("fake"))
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev.GetVFID()).To(Equal(0))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("device's PF name is not available", func() {
			It("device should be added", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{"sys/bus/pci/devices/0000:00:00.1"},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
				}
				defer fs.Use()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{
					Address: "0000:00:00.1",
					Vendor:  &pcidb.Vendor{},
					Product: &pcidb.Product{},
				}
				rc := &types.ResourceConfig{}
				infoProviders := make([]types.DeviceInfoProvider, 0)

				dev, err := devices.NewPciDevice(in, f, rc, infoProviders)

				Expect(dev).NotTo(BeNil())
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
