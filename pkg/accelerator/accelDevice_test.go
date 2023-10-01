// Copyright 2020 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package accelerator_test

import (
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/accelerator"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Accelerator", func() {
	Describe("creating new accelerator device", func() {
		pciAddr := "0000:00:00.1"
		newPciDeviceFn := func() *ghw.PCIDevice {
			return &ghw.PCIDevice{
				Address: pciAddr,
				Vendor:  &pcidb.Vendor{ID: ""},
				Product: &pcidb.Product{ID: ""},
			}
		}
		Context("successfully", func() {
			It("should return AccelDevice object instance", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
					Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0")},
				}
				defer fs.Use()()

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn()
				config := &types.ResourceConfig{}

				out, err := accelerator.NewAccelDevice(in, f, config)

				// TODO: assert other fields once implemented
				Expect(out.GetDriver()).To(Equal("vfio-pci"))
				Expect(out.GetDeviceSpecs()).To(HaveLen(2)) // /dev/vfio/vfio0 and default /dev/vfio/vfio
				envs := out.GetEnvVal()
				Expect(len(envs)).To(Equal(2))

				vfioMap, exist := envs["vfio"]
				Expect(exist).To(BeTrue())
				Expect(len(vfioMap)).To(Equal(2))
				vfio, exist := envs["vfio"]["mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/vfio"))
				vfio, exist = envs["vfio"]["dev-mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/0"))
				genericMap, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				Expect(len(genericMap)).To(Equal(1))
				generic, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(generic).To(Equal("0000:00:00.1"))

				Expect(out.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not populate topology due to negative numa_node", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
					Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("-1")},
				}
				defer fs.Use()()

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn()
				config := &types.ResourceConfig{}

				out, err := accelerator.NewAccelDevice(in, f, config)

				Expect(out.GetAPIDevice().Topology).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not populate topology due to missing numa_node", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
				}
				defer fs.Use()()

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn()
				config := &types.ResourceConfig{}

				out, err := accelerator.NewAccelDevice(in, f, config)

				Expect(out.GetAPIDevice().Topology).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not populate topology due to config excluding topology being set", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
					Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("1")},
				}
				defer fs.Use()()

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn()
				config := &types.ResourceConfig{ExcludeTopology: true}

				out, err := accelerator.NewAccelDevice(in, f, config)

				Expect(out.GetAPIDevice().Topology).To(BeNil())
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("cannot get device's driver", func() {
			It("should fail", func() {
				fs := &utils.FakeFilesystem{
					Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
					Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/driver": []byte("not a symlink")},
				}
				defer fs.Use()()

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn()
				config := &types.ResourceConfig{}

				dev, err := accelerator.NewAccelDevice(in, f, config)

				Expect(dev).To(BeNil())
				Expect(err).To(HaveOccurred())
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

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn()
				config := &types.ResourceConfig{}

				dev, err := accelerator.NewAccelDevice(in, f, config)
				Expect(err).NotTo(HaveOccurred())
				Expect(dev).NotTo(BeNil())

				envs := dev.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
				genericMap, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				Expect(len(genericMap)).To(Equal(1))
				generic, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(generic).To(Equal("0000:00:00.1"))
			})
		})
	})
})
