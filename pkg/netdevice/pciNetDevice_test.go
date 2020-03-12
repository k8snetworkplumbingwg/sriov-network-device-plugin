package netdevice_test

import (
	"testing"

	"github.com/jaypipes/ghw"

	"github.com/intel/sriov-network-device-plugin/pkg/factory"
	"github.com/intel/sriov-network-device-plugin/pkg/netdevice"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNetdevice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Netdevice Suite")
}

var _ = Describe("PciNetDevice", func() {
	Describe("creating new PciNetDevice", func() {
		Context("succesfully", func() {
			It("should populate fields", func() {
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

				dev, err := netdevice.NewPciNetDevice(in, f)

				Expect(dev.GetDriver()).To(Equal("vfio-pci"))
				Expect(dev.GetNetName()).To(Equal("eth0"))
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(2)) // /dev/vfio/vfio0 and default /dev/vfio/vfio
				Expect(dev.GetRdmaSpec().IsRdma()).To(BeFalse())
				Expect(dev.GetRdmaSpec().GetRdmaDeviceSpec()).To(HaveLen(0))
				Expect(dev.GetLinkType()).To(Equal("fakeLinkType"))
				Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(dev.GetNumaInfo()).To(Equal("0"))
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

				dev, err := netdevice.NewPciNetDevice(in, f)

				Expect(dev.GetAPIDevice().Topology).To(BeNil())
				Expect(dev.GetNumaInfo()).To(Equal(""))
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

				dev, err := netdevice.NewPciNetDevice(in, f)

				Expect(dev.GetAPIDevice().Topology).To(BeNil())
				Expect(dev.GetNumaInfo()).To(Equal(""))
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

				dev, err := netdevice.NewPciNetDevice(in, f)

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

				dev, err := netdevice.NewPciNetDevice(in, f)

				Expect(dev).NotTo(BeNil())
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
