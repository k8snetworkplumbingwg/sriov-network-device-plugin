package resources

import (
	"github.com/jaypipes/ghw"

	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PciNetDevice", func() {
	Describe("creating new PciNetDevice", func() {
		Context("succesfully", func() {
			It("should populate fields", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
				}
				defer fs.Use()()

				f := NewResourceFactory("fake", "fake")
				in := &ghw.PCIDevice{Address: "0000:00:00.1"}

				dev, err := NewPciNetDevice(in, f)
				out := dev.(*pciNetDevice)

				// TODO: assert other fields once implemented
				Expect(out.driver).To(Equal("vfio-pci"))
				Expect(out.env).To(Equal("0000:00:00.1"))
				Expect(out.deviceSpecs).To(HaveLen(2)) // /dev/vfio/vfio0 and default /dev/vfio/vfio

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

				f := NewResourceFactory("fake", "fake")
				in := &ghw.PCIDevice{
					Address: "0000:00:00.1",
				}

				dev, err := NewPciNetDevice(in, f)

				Expect(dev).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
