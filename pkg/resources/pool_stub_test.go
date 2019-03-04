package resources

import (
	"github.com/jaypipes/ghw"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PoolStub", func() {
	var (
		fs     *utils.FakeFilesystem
		f      types.ResourceFactory
		devs   []string
		d1, d2 types.PciNetDevice
		rp     types.ResourcePool
	)
	BeforeEach(func() {
		fs = &utils.FakeFilesystem{
			Dirs: []string{
				"sys/bus/pci/devices/0000:00:00.1",
				"sys/bus/pci/devices/0000:00:00.2",
				"sys/kernel/iommu_groups/0",
				"sys/kernel/iommu_groups/1",
				"sys/bus/pci/drivers/vfio-pci",
			},
			Symlinks: map[string]string{
				"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
				"sys/bus/pci/devices/0000:00:00.2/iommu_group": "../../../../kernel/iommu_groups/1",
				"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
				"sys/bus/pci/devices/0000:00:00.2/driver":      "../../../../bus/pci/drivers/vfio-pci",
			},
		}
		f = NewResourceFactory("fake", "fake")
		devs = []string{"0000:00:00.1", "0000:00:00.2"}
	})
	Describe("getting device specs", func() {
		Context("for valid devices", func() {
			It("should return valid DeviceSpec array", func() {
				defer fs.Use()()

				d1, _ = NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.1"}, f)
				d2, _ = NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.2"}, f)
				rp = &resourcePool{
					devicePool: map[string]types.PciNetDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				}
				specs := rp.GetDeviceSpecs(devs)

				expected := []*pluginapi.DeviceSpec{
					{ContainerPath: "/dev/vfio/vfio", HostPath: "/dev/vfio/vfio", Permissions: "mrw"},
					{ContainerPath: "/dev/vfio/0", HostPath: "/dev/vfio/0", Permissions: "mrw"},
					{ContainerPath: "/dev/vfio/1", HostPath: "/dev/vfio/1", Permissions: "mrw"},
				}
				Expect(specs).To(HaveLen(3))
				Expect(specs).To(ConsistOf(expected))
			})
		})
	})
	Describe("getting envs", func() {
		Context("for valid devices", func() {
			It("should return valid envs array", func() {
				defer fs.Use()()

				d1, _ = NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.1"}, f)
				d2, _ = NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.2"}, f)
				rp = &resourcePool{
					devicePool: map[string]types.PciNetDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				}
				envs := rp.GetEnvs(devs)

				expected := []string{"0000:00:00.1", "0000:00:00.2"}
				Expect(envs).To(HaveLen(2))
				Expect(envs).To(ConsistOf(expected))
			})
		})
	})
	Describe("getting mounts", func() {
		Context("for valid devices", func() {
			It("should return valid mounts array", func() {
				defer fs.Use()()

				d1, _ = NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.1"}, f)
				d2, _ = NewPciNetDevice(&ghw.PCIDevice{Address: "0000:00:00.2"}, f)
				rp = &resourcePool{
					devicePool: map[string]types.PciNetDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				}
				mounts := rp.GetMounts(devs)

				// in current implementation vfio and others return empty arrays
				expected := []*pluginapi.Mount{}
				Expect(mounts).To(HaveLen(0))
				Expect(mounts).To(ConsistOf(expected))
			})

		})
	})
})
