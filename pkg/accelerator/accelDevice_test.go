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

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/accelerator"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Accelerator", func() {
	Describe("creating new accelerator device", func() {
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
				defer utils.UseFakeLinks()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{Address: "0000:00:00.1"}

				out, err := accelerator.NewAccelDevice(in, f)

				// TODO: assert other fields once implemented
				Expect(out.GetDriver()).To(Equal("vfio-pci"))
				Expect(out.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(out.GetDeviceSpecs()).To(HaveLen(2)) // /dev/vfio/vfio0 and default /dev/vfio/vfio
				Expect(out.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(out.GetNumaInfo()).To(Equal("0"))
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
				defer utils.UseFakeLinks()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{Address: "0000:00:00.1"}

				out, err := accelerator.NewAccelDevice(in, f)

				Expect(out.GetAPIDevice().Topology).To(BeNil())
				Expect(out.GetNumaInfo()).To(Equal(""))
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
				defer utils.UseFakeLinks()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{Address: "0000:00:00.1"}

				out, err := accelerator.NewAccelDevice(in, f)

				Expect(out.GetAPIDevice().Topology).To(BeNil())
				Expect(out.GetNumaInfo()).To(Equal(""))
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
				defer utils.UseFakeLinks()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{
					Address: "0000:00:00.1",
				}

				dev, err := accelerator.NewAccelDevice(in, f)

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
				defer utils.UseFakeLinks()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{
					Address: "0000:00:00.1",
				}

				dev, err := accelerator.NewAccelDevice(in, f)
				Expect(err).NotTo(HaveOccurred())
				Expect(dev).NotTo(BeNil())
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
			})
		})
	})
})
