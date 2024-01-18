package resources_test

import (
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResourcePool", func() {
	var (
		fs     *utils.FakeFilesystem
		f      types.ResourceFactory
		devs   []string
		d1, d2 types.PciNetDevice
		rp     types.ResourcePool
		rc     *types.ResourceConfig
	)
	newPciDeviceFn := func(pciAddr string) *ghw.PCIDevice {
		return &ghw.PCIDevice{
			Address: pciAddr,
			Vendor:  &pcidb.Vendor{ID: ""},
			Product: &pcidb.Product{ID: ""},
		}
	}

	BeforeEach(func() {
		fs = &utils.FakeFilesystem{
			Dirs: []string{
				"sys/bus/pci/devices/0000:00:00.1/net/enp2s0f0v0",
				"sys/bus/pci/devices/0000:01:00.0/net/enp2s0f0",
				"sys/bus/pci/devices/0000:00:00.2/net/enp2s0f1v0",
				"sys/kernel/iommu_groups/0",
				"sys/kernel/iommu_groups/1",
				"sys/bus/pci/drivers/vfio-pci",
			},
			Symlinks: map[string]string{
				"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
				"sys/bus/pci/devices/0000:00:00.2/iommu_group": "../../../../kernel/iommu_groups/1",
				"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
				"sys/bus/pci/devices/0000:00:00.2/driver":      "../../../../bus/pci/drivers/vfio-pci",
				"sys/bus/pci/devices/0000:00:00.1/physfn":      "../0000:01:00.0",
			},
		}
		f = factory.NewResourceFactory("fake", "fake", true, false)
		rc = &types.ResourceConfig{SelectorObjs: []interface{}{types.NetDeviceSelectors{}}}
		devs = []string{"0000:00:00.1", "0000:00:00.2"}
	})
	Describe("getting device specs", func() {
		Context("for valid devices", func() {
			It("should return valid DeviceSpec array", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				d1, _ = netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.1"), f, rc, 0)
				d2, _ = netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.2"), f, rc, 0)
				rp = resources.NewResourcePool(rc,
					map[string]types.HostDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				)
				specs := rp.GetDeviceSpecs(devs)

				expected := []*pluginapi.DeviceSpec{
					{ContainerPath: "/dev/vfio/vfio", HostPath: "/dev/vfio/vfio", Permissions: "rw"},
					{ContainerPath: "/dev/vfio/0", HostPath: "/dev/vfio/0", Permissions: "rw"},
					{ContainerPath: "/dev/vfio/1", HostPath: "/dev/vfio/1", Permissions: "rw"},
				}
				Expect(specs).To(HaveLen(3))
				Expect(specs).To(ConsistOf(expected))
			})
		})
	})
	Describe("getting mounts", func() {
		Context("for valid devices", func() {
			It("should return valid mounts array", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				d1, _ = netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.1"), f, rc, 0)
				d2, _ = netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.2"), f, rc, 0)
				rp = resources.NewResourcePool(rc,
					map[string]types.HostDevice{
						"0000:00:00.1": d1,
						"0000:00:00.2": d2,
					},
				)
				mounts := rp.GetMounts(devs)

				// in current implementation vfio and others return empty arrays
				expected := []*pluginapi.Mount{}
				Expect(mounts).To(HaveLen(0))
				Expect(mounts).To(ConsistOf(expected))
			})

		})
	})
	Describe("GetDevices", func() {
		It("Returns API devices for PCIDevices in the pool", func() {
			defer fs.Use()()
			utils.SetDefaultMockNetlinkProvider()

			d1, _ = netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.1"), f, rc, 0)
			d2, _ = netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.2"), f, rc, 0)
			rp = resources.NewResourcePool(rc,
				map[string]types.HostDevice{
					"0000:00:00.1": d1,
					"0000:00:00.2": d2,
				},
			)
			devices := rp.GetDevices()
			Expect(devices).To(HaveLen(2))
			Expect(devices).To(HaveKey("0000:00:00.1"))
			Expect(devices).To(HaveKey("0000:00:00.2"))
			Expect(devices["0000:00:00.1"].ID).To(Equal("0000:00:00.1"))
			Expect(devices["0000:00:00.2"].ID).To(Equal("0000:00:00.2"))
		})
	})
})
